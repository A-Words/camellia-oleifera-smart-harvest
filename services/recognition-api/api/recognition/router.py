from __future__ import annotations

import asyncio
import time

import numpy as np
from fastapi import APIRouter, File, HTTPException, Request, UploadFile, WebSocket, WebSocketDisconnect

from schemas.api import (
    CurrentModelResponse,
    ImageInferResponse,
    StreamFrameEnvelope,
    StreamSummaryEnvelope,
)

router = APIRouter(tags=["recognition"])


def _decode_image_bytes(data: bytes) -> np.ndarray:
    try:
        import cv2
    except Exception as exc:
        raise RuntimeError('opencv-python-headless is required for image decoding') from exc

    arr = np.frombuffer(data, dtype=np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError('Invalid image bytes')
    return img


@router.get('/current', response_model=CurrentModelResponse)
async def current_model(request: Request) -> CurrentModelResponse:
    meta = request.app.state.pipeline.model_meta()
    return CurrentModelResponse(**meta.model_dump())


@router.post('/image', response_model=ImageInferResponse)
async def infer_image(request: Request, file: UploadFile = File(...)) -> ImageInferResponse:
    pipeline = request.app.state.pipeline
    service_cfg = request.app.state.service_cfg

    body = await file.read()
    max_bytes = service_cfg.max_upload_mb * 1024 * 1024
    if len(body) > max_bytes:
        raise HTTPException(status_code=413, detail='Uploaded file is too large')

    try:
        started = time.perf_counter()
        frame = _decode_image_bytes(body)
        result, _ = pipeline.infer_image(frame)
        result = await pipeline.enrich_image_result(frame, result)
        inference_ms = (time.perf_counter() - started) * 1000.0
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except RuntimeError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc

    meta = pipeline.model_meta()
    return ImageInferResponse(
        model_version=meta.model_version,
        schema_version=meta.schema_version,
        inference_ms=inference_ms,
        result=result,
    )


@router.websocket('/stream')
async def infer_stream(websocket: WebSocket) -> None:
    await websocket.accept()
    pipeline = websocket.app.state.pipeline
    session = pipeline.create_stream_session()
    started = time.perf_counter()
    pending_tasks: set[asyncio.Task[None]] = set()

    try:
        while True:
            msg = await websocket.receive()
            msg_type = msg.get('type')
            if msg_type == 'websocket.disconnect':
                break

            if 'text' in msg and msg['text']:
                text = msg['text'].strip().lower()
                if text in {'close', 'stop', 'eos'}:
                    break
                await websocket.send_json({'type': 'error', 'detail': 'Unsupported text command'})
                continue

            payload = msg.get('bytes')
            if not payload:
                await websocket.send_json({'type': 'error', 'detail': 'Empty frame payload'})
                continue

            timestamp_ms = int((time.perf_counter() - started) * 1000)
            try:
                frame = _decode_image_bytes(payload)
                result = pipeline.infer_stream_frame(frame, session, timestamp_ms)
                requests = pipeline.build_stream_ripeness_requests(frame, result, session)
            except Exception as exc:
                await websocket.send_json({'type': 'error', 'detail': str(exc)})
                continue

            meta = pipeline.model_meta()
            envelope = StreamFrameEnvelope(
                model_version=meta.model_version,
                schema_version=meta.schema_version,
                result=result,
            )
            await websocket.send_json(envelope.model_dump())
            if requests:
                task = asyncio.create_task(pipeline.resolve_stream_ripeness_requests(requests, session))
                pending_tasks.add(task)
                task.add_done_callback(lambda finished: pending_tasks.discard(finished))
    except WebSocketDisconnect:
        pass
    finally:
        if pending_tasks:
            for task in list(pending_tasks):
                task.cancel()
            await asyncio.gather(*pending_tasks, return_exceptions=True)
        meta = pipeline.model_meta()
        summary = session.aggregator.build_summary()
        envelope = StreamSummaryEnvelope(
            model_version=meta.model_version,
            schema_version=meta.schema_version,
            summary=summary,
        )
        try:
            await websocket.send_json(envelope.model_dump())
        except Exception:
            pass
