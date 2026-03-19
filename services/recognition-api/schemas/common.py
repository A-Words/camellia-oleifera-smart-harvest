from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


class Detection(BaseModel):
    bbox: tuple[float, float, float, float]
    class_name: Literal["camellia_oleifera_fruit"] = "camellia_oleifera_fruit"
    confidence: float = Field(ge=0.0, le=1.0)
    track_id: int | None = None


class FrameSummary(BaseModel):
    total: int = 0


class FrameResult(BaseModel):
    frame_index: int = Field(ge=0)
    timestamp_ms: int = Field(ge=0)
    detections: list[Detection]
    frame_summary: FrameSummary


class SessionSummary(BaseModel):
    total_detected: int = 0


class ModelMeta(BaseModel):
    model_version: str
    schema_version: str
    adapter: str
    loaded: bool
