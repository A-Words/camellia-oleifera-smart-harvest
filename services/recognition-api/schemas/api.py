from __future__ import annotations

from pydantic import BaseModel

from schemas.common import FrameResult, ModelMeta, SessionSummary


class ImageInferResponse(BaseModel):
    model_version: str
    schema_version: str
    inference_ms: float
    result: FrameResult


class CurrentModelResponse(BaseModel):
    model_version: str
    schema_version: str
    adapter: str
    loaded: bool
    load_error: str | None = None
    ripeness_enabled: bool = False
    ripeness_adapter: str | None = None
    ripeness_loaded: bool = False
    ripeness_error: str | None = None


class StreamFrameEnvelope(BaseModel):
    type: str = "frame"
    model_version: str
    schema_version: str
    result: FrameResult


class StreamSummaryEnvelope(BaseModel):
    type: str = "summary"
    model_version: str
    schema_version: str
    summary: SessionSummary
