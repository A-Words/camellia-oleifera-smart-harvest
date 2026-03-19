from __future__ import annotations

import importlib

import numpy as np
import pytest
from fastapi.testclient import TestClient

from core.recognition.adapters.base import DetectorAdapter, RawDetection
from core.recognition.pipeline import InferencePipeline
from main import app


class FakeDetector(DetectorAdapter):
    name = 'fake'

    def __init__(self) -> None:
        self._loaded = True

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        self._loaded = True

    def warmup(self) -> None:
        return

    def predict(self, frame: np.ndarray):
        return [RawDetection(bbox=(1, 1, 20, 20), class_id=1, confidence=0.95)]

    def ripeness_from_class_id(self, class_id: int) -> str:
        return 'half'


@pytest.mark.perf
def test_stream_contract(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((120, 120, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(FakeDetector(), model_version='1.0.0', schema_version='v1')

        with client.websocket_connect('/v1/recognition/stream') as ws:
            for _ in range(3):
                ws.send_bytes(b'frame-bytes')
                msg = ws.receive_json()
                assert msg['type'] == 'frame'
            ws.send_text('eos')
            summary = ws.receive_json()
            assert summary['type'] == 'summary'
