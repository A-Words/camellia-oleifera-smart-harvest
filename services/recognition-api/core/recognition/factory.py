from __future__ import annotations

from core.recognition.adapters.base import DetectorAdapter
from core.recognition.adapters.yolo_stable import YoloStableAdapter
from core.recognition.ripeness import (
    BrokenRipenessClassifier,
    OpenAICompatibleRipenessClassifier,
    RipenessClassifier,
)
from settings import ModelConfig


def build_detector(cfg: ModelConfig) -> DetectorAdapter:
    return YoloStableAdapter(cfg)


def build_ripeness_classifier(cfg: ModelConfig) -> RipenessClassifier | None:
    if not cfg.ripeness_enabled:
        return None

    adapter = cfg.ripeness_adapter.strip().lower()
    if adapter in {"openai_compatible", "openai_compatible_vlm"}:
        return OpenAICompatibleRipenessClassifier(cfg)

    return BrokenRipenessClassifier(
        name=cfg.ripeness_adapter or "unknown",
        load_error=f"Unsupported ripeness adapter: {cfg.ripeness_adapter}",
        crop_padding_ratio=cfg.vlm_crop_padding_ratio,
    )
