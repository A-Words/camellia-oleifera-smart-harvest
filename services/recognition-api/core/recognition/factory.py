from __future__ import annotations

from core.recognition.adapters.base import DetectorAdapter
from core.recognition.adapters.yolo_stable import YoloStableAdapter
from settings import ModelConfig


def build_detector(cfg: ModelConfig) -> DetectorAdapter:
    return YoloStableAdapter(cfg)
