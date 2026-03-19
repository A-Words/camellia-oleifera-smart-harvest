from __future__ import annotations

import os
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI

from api.recognition.router import router as recognition_router
from api.system.router import router as system_router
from core.recognition.factory import build_detector
from core.recognition.pipeline import InferencePipeline
from settings import (
    ServiceConfig,
    load_model_config,
    load_service_config,
)

_REPO_ROOT = Path(__file__).resolve().parents[2]


def _resolve_config_path(raw_path: str) -> Path:
    candidate = Path(raw_path).expanduser()
    if candidate.is_absolute():
        return candidate

    cwd_candidate = Path.cwd() / candidate
    if cwd_candidate.exists():
        return cwd_candidate.resolve()

    repo_candidate = _REPO_ROOT / candidate
    if repo_candidate.exists():
        return repo_candidate.resolve()

    return repo_candidate


def _ensure_config_file(path: Path, env_var: str) -> None:
    if path.exists():
        return

    example_path = Path(f"{path}.example")
    if example_path.exists():
        hint = f" Copy '{example_path.as_posix()}' to '{path.as_posix()}'."
    else:
        hint = ""

    raise RuntimeError(
        f"Missing config file: '{path.as_posix()}'. "
        f"Set {env_var} to an existing file.{hint}"
    )


@asynccontextmanager
async def lifespan(app: FastAPI):
    model_cfg_path = _resolve_config_path(
        os.getenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml')
    )
    service_cfg_path = _resolve_config_path(
        os.getenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml')
    )
    _ensure_config_file(model_cfg_path, "CAMELLIA_RECOGNITION_CONFIG")
    _ensure_config_file(service_cfg_path, "CAMELLIA_SERVICE_CONFIG")

    model_cfg = load_model_config(model_cfg_path)
    service_cfg: ServiceConfig = load_service_config(service_cfg_path)

    detector = build_detector(model_cfg)
    try:
        detector.load()
        detector.warmup()
    except Exception:
        # Keep service booted in degraded mode for health visibility.
        pass

    app.state.service_cfg = service_cfg
    app.state.pipeline = InferencePipeline(
        detector=detector,
        model_version=model_cfg.model_version,
        schema_version=service_cfg.schema_version,
    )

    yield


app = FastAPI(
    title='camellia-oleifera-recognition-api',
    version='0.1.0',
    lifespan=lifespan,
)
app.include_router(system_router, prefix='/v1')
app.include_router(recognition_router, prefix='/v1/recognition')
