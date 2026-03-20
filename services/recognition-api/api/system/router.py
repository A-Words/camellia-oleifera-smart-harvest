from __future__ import annotations

from fastapi import APIRouter, Request

from schemas.system import HealthResponse

router = APIRouter(tags=["system"])


@router.get('/health', response_model=HealthResponse)
async def health(request: Request) -> HealthResponse:
    pipeline = request.app.state.pipeline
    meta = pipeline.model_meta()
    ripeness_ready = (not meta.ripeness_enabled) or meta.ripeness_loaded
    status = 'ok' if meta.loaded and ripeness_ready else 'degraded'
    return HealthResponse(status=status, model=meta)
