from __future__ import annotations

import re
from pathlib import Path

from pydantic import BaseModel, Field

try:
    import yaml  # type: ignore
except Exception:  # pragma: no cover
    yaml = None

DEFAULT_SCHEMA_VERSION = "v1"
_CUDA_DEVICE_PATTERN = re.compile(r"^\d+(,\d+)*$")
_REPO_ROOT = Path(__file__).resolve().parents[2]


class ModelConfig(BaseModel):
    yolo_version: str = "yolo26n"
    model_version: str = "1.0.0"
    model_path: str = ""
    conf_threshold: float = Field(default=0.25, ge=0.0, le=1.0)
    nms_iou: float = Field(default=0.45, ge=0.0, le=1.0)
    device: str = "auto"
    ripeness_enabled: bool = False
    ripeness_adapter: str = "openai_compatible"
    vlm_base_url: str = "https://api.openai.com/v1"
    vlm_model: str = ""
    vlm_timeout_s: float = Field(default=15.0, gt=0.0)
    vlm_batch_size: int = Field(default=4, ge=1)
    vlm_crop_padding_ratio: float = Field(default=0.12, ge=0.0, le=0.5)


class ServiceConfig(BaseModel):
    app_name: str = "camellia-oleifera-recognition-api"
    api_prefix: str = "/v1"
    schema_version: str = DEFAULT_SCHEMA_VERSION
    max_upload_mb: int = 10


def _torch_cuda_runtime() -> tuple[bool, int]:
    try:
        import torch
    except Exception:
        return False, 0

    try:
        return bool(torch.cuda.is_available()), int(torch.cuda.device_count())
    except Exception:
        return False, 0


def _requests_cuda(device: str) -> bool:
    normalized = device.strip().lower()
    if not normalized or normalized in {"cpu", "mps"}:
        return False
    if normalized.startswith("cuda"):
        return True
    return bool(_CUDA_DEVICE_PATTERN.fullmatch(normalized))


def resolve_torch_device(device: str) -> tuple[str, str | None]:
    requested = (device or "").strip() or "cpu"
    cuda_available, cuda_device_count = _torch_cuda_runtime()

    if requested.lower() == "auto":
        if cuda_available and cuda_device_count > 0:
            return "cuda:0", None
        return (
            "cpu",
            (
                "Requested device='auto' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    if _requests_cuda(requested) and (not cuda_available or cuda_device_count <= 0):
        return (
            "cpu",
            (
                f"Requested device='{requested}' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    return requested, None


def _parse_simple_yaml(text: str) -> dict:
    data: dict[str, object] = {}
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith('#') or ':' not in line:
            continue
        key, value = line.split(':', 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        low = value.lower()
        if low in {'true', 'false'}:
            data[key] = low == 'true'
            continue
        try:
            if '.' in value:
                data[key] = float(value)
            else:
                data[key] = int(value)
            continue
        except ValueError:
            data[key] = value
    return data


def _load_yaml(path: Path) -> dict:
    if not path.exists():
        return {}
    text = path.read_text(encoding='utf-8')
    if yaml is not None:
        data = yaml.safe_load(text) or {}
        if not isinstance(data, dict):
            raise ValueError(f"Config file must contain a YAML mapping: {path}")
        return data
    return _parse_simple_yaml(text)


def _resolve_model_path(raw_path: str, config_path: Path) -> str:
    normalized = (raw_path or "").strip()
    if not normalized:
        return ""

    candidate = Path(normalized).expanduser()
    if candidate.is_absolute():
        return str(candidate)

    config_relative = (config_path.parent / candidate).resolve()
    if config_relative.exists():
        return str(config_relative)

    repo_relative = (_REPO_ROOT / candidate).resolve()
    if repo_relative.exists():
        return str(repo_relative)

    return str(repo_relative)


def load_model_config(path: Path) -> ModelConfig:
    data = _load_yaml(path)
    if "model_path" in data and isinstance(data["model_path"], str):
        data["model_path"] = _resolve_model_path(data["model_path"], path)
    return ModelConfig.model_validate(data)


def load_service_config(path: Path) -> ServiceConfig:
    return ServiceConfig.model_validate(_load_yaml(path))
