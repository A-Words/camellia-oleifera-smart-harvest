from __future__ import annotations

from fastapi import APIRouter, Request

from schemas.system import HealthResponse

router = APIRouter(tags=["system"])


@router.get('/health', response_model=HealthResponse)
async def health(request: Request) -> HealthResponse:
    pipeline = request.app.state.pipeline
    meta = pipeline.model_meta()
    status = 'ok' if meta.loaded else 'degraded'
    return HealthResponse(status=status, model=meta)
