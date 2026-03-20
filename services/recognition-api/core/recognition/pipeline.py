from __future__ import annotations

import time
from dataclasses import dataclass, field

import numpy as np

from core.recognition.adapters.base import DetectorAdapter
from core.recognition.aggregator import SessionAggregator
from core.recognition.ripeness import RipenessClassifier, crop_detection_regions
from core.recognition.tracker import ByteTrackManager
from schemas.common import Detection, FrameResult, ModelMeta, RipenessLabel


def _sanitize_bbox(bbox: tuple[float, float, float, float], width: int, height: int) -> tuple[float, float, float, float]:
    x1, y1, x2, y2 = bbox
    x1 = max(0.0, min(float(width - 1), x1))
    y1 = max(0.0, min(float(height - 1), y1))
    x2 = max(0.0, min(float(width - 1), x2))
    y2 = max(0.0, min(float(height - 1), y2))
    if x2 < x1:
        x1, x2 = x2, x1
    if y2 < y1:
        y1, y2 = y2, y1
    return (x1, y1, x2, y2)


@dataclass
class StreamSession:
    tracker: ByteTrackManager
    aggregator: SessionAggregator
    frame_index: int = 0
    ripeness_by_track_id: dict[int, RipenessLabel | None] = field(default_factory=dict)
    ripeness_pending_track_ids: set[int] = field(default_factory=set)

    def prune_ripeness_cache(self, active_track_ids: set[int]) -> None:
        self.ripeness_by_track_id = {
            track_id: ripeness
            for track_id, ripeness in self.ripeness_by_track_id.items()
            if track_id in active_track_ids
        }
        self.ripeness_pending_track_ids = {
            track_id
            for track_id in self.ripeness_pending_track_ids
            if track_id in active_track_ids
        }


@dataclass(slots=True)
class StreamRipenessRequest:
    track_id: int
    crop: np.ndarray


class InferencePipeline:
    def __init__(
        self,
        detector: DetectorAdapter,
        model_version: str,
        schema_version: str,
        ripeness_classifier: RipenessClassifier | None = None,
    ) -> None:
        self.detector = detector
        self.model_version = model_version
        self.schema_version = schema_version
        self.ripeness_classifier = ripeness_classifier

    def model_meta(self) -> ModelMeta:
        return ModelMeta(
            model_version=self.model_version,
            schema_version=self.schema_version,
            adapter=self.detector.name,
            loaded=self.detector.loaded,
            load_error=getattr(self.detector, "load_error", None),
            ripeness_enabled=self.ripeness_classifier is not None,
            ripeness_adapter=self.ripeness_classifier.name if self.ripeness_classifier is not None else None,
            ripeness_loaded=bool(self.ripeness_classifier and self.ripeness_classifier.loaded),
            ripeness_error=getattr(self.ripeness_classifier, "load_error", None) if self.ripeness_classifier is not None else None,
        )

    def create_stream_session(self) -> StreamSession:
        return StreamSession(tracker=ByteTrackManager(), aggregator=SessionAggregator())

    def infer_image(self, frame: np.ndarray) -> tuple[FrameResult, float]:
        session = self.create_stream_session()
        start = time.perf_counter()
        result = self._infer_frame(frame, session, timestamp_ms=0, use_track=False)
        elapsed_ms = (time.perf_counter() - start) * 1000.0
        return result, elapsed_ms

    def infer_stream_frame(self, frame: np.ndarray, session: StreamSession, timestamp_ms: int) -> FrameResult:
        return self._infer_frame(frame, session, timestamp_ms=timestamp_ms, use_track=True)

    async def enrich_image_result(self, frame: np.ndarray, result: FrameResult) -> FrameResult:
        if not self._ripeness_ready() or not result.detections:
            return result

        crops = crop_detection_regions(
            frame,
            result.detections,
            self.ripeness_classifier.crop_padding_ratio,
        )
        if not crops:
            return result

        ripeness_values = await self.ripeness_classifier.classify_crops([item.crop for item in crops])
        for crop, ripeness in zip(crops, ripeness_values):
            result.detections[crop.detection_index].ripeness = ripeness
        return result

    def build_stream_ripeness_requests(
        self,
        frame: np.ndarray,
        result: FrameResult,
        session: StreamSession,
    ) -> list[StreamRipenessRequest]:
        if not self._ripeness_ready() or not result.detections:
            return []

        requests: list[StreamRipenessRequest] = []
        crops = crop_detection_regions(
            frame,
            result.detections,
            self.ripeness_classifier.crop_padding_ratio,
        )
        for crop in crops:
            track_id = crop.track_id
            if track_id is None:
                continue
            if track_id in session.ripeness_by_track_id:
                continue
            if track_id in session.ripeness_pending_track_ids:
                continue
            session.ripeness_pending_track_ids.add(track_id)
            requests.append(StreamRipenessRequest(track_id=track_id, crop=crop.crop))
        return requests

    async def resolve_stream_ripeness_requests(
        self,
        requests: list[StreamRipenessRequest],
        session: StreamSession,
    ) -> None:
        if not requests:
            return

        ripeness_values: list[RipenessLabel | None]
        try:
            if not self._ripeness_ready():
                ripeness_values = [None] * len(requests)
            else:
                ripeness_values = await self.ripeness_classifier.classify_crops([item.crop for item in requests])
        except Exception:
            ripeness_values = [None] * len(requests)

        if len(ripeness_values) < len(requests):
            ripeness_values.extend([None] * (len(requests) - len(ripeness_values)))

        for request, ripeness in zip(requests, ripeness_values):
            session.ripeness_by_track_id[request.track_id] = ripeness
            session.ripeness_pending_track_ids.discard(request.track_id)

    def _infer_frame(self, frame: np.ndarray, session: StreamSession, timestamp_ms: int, use_track: bool) -> FrameResult:
        if frame.ndim != 3:
            raise ValueError("Expected BGR frame with shape [H, W, C]")
        if not self.detector.loaded:
            raise RuntimeError(getattr(self.detector, "load_error", None) or "Detector is not loaded")

        raw_dets = list(self.detector.predict(frame))
        height, width = frame.shape[:2]
        if use_track:
            tracked = session.tracker.update(raw_dets)
        else:
            tracked = []

        track_map = {id(t.det): t.track_id for t in tracked}
        active_track_ids = session.tracker.active_track_ids() if use_track else set()
        detections: list[Detection] = []
        track_ids: list[int | None] = []

        for det in raw_dets:
            sanitized_bbox = _sanitize_bbox(det.bbox, width, height)
            track_id = track_map.get(id(det)) if use_track else None
            detections.append(
                Detection(
                    bbox=sanitized_bbox,
                    class_name=self.detector.class_name_from_class_id(det.class_id),
                    confidence=det.confidence,
                    track_id=track_id,
                    ripeness=session.ripeness_by_track_id.get(track_id) if track_id is not None else None,
                )
            )
            track_ids.append(track_id)

        session.aggregator.update_session(track_ids)
        if use_track:
            session.prune_ripeness_cache(active_track_ids)
        frame_summary = session.aggregator.frame_summary(len(detections))

        result = FrameResult(
            frame_index=session.frame_index,
            timestamp_ms=timestamp_ms,
            detections=detections,
            frame_summary=frame_summary,
        )
        session.frame_index += 1
        return result

    def _ripeness_ready(self) -> bool:
        return self.ripeness_classifier is not None and self.ripeness_classifier.loaded
