from __future__ import annotations

from pydantic import BaseModel

from schemas.common import ModelMeta


class HealthResponse(BaseModel):
    status: str
    model: ModelMeta
