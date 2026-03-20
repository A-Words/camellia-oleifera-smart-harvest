from __future__ import annotations

import importlib
import time

import numpy as np
from fastapi.testclient import TestClient

from core.recognition.adapters.base import DetectorAdapter, RawDetection
from core.recognition.pipeline import InferencePipeline
from core.recognition.ripeness.base import RipenessClassifier
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
        return [RawDetection(bbox=(5, 5, 30, 30), class_id=2, confidence=0.8)]

    def class_name_from_class_id(self, class_id: int) -> str:
        return 'camellia_oleifera_fruit'


class BrokenDetector(DetectorAdapter):
    name = 'broken'

    def __init__(self, load_error: str) -> None:
        self._loaded = False
        self.load_error = load_error

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        self._loaded = False

    def warmup(self) -> None:
        return

    def predict(self, frame: np.ndarray):
        raise RuntimeError(self.load_error)

    def class_name_from_class_id(self, class_id: int) -> str:
        return 'camellia_oleifera_fruit'


class FakeRipenessClassifier(RipenessClassifier):
    name = 'fake_vlm'

    def __init__(self, outputs: list[str | None], load_error: str | None = None) -> None:
        self.outputs = outputs
        self.crop_padding_ratio = 0.12
        self.load_error = load_error
        self._loaded = load_error is None

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        return

    async def classify_crops(self, crops):
        return self.outputs[:len(crops)]


def test_health_and_image_infer(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(FakeDetector(), model_version='1.0.0', schema_version='v1')

        health = client.get('/v1/health')
        assert health.status_code == 200
        assert health.json()['status'] == 'ok'

        resp = client.post('/v1/recognition/image', files={'file': ('x.jpg', b'raw-bytes', 'image/jpeg')})
        assert resp.status_code == 200
        body = resp.json()
        assert body['result']['frame_summary']['total'] == 1
        assert body['result']['detections'][0]['class_name'] == 'camellia_oleifera_fruit'
        assert body['result']['detections'][0]['ripeness'] is None


def test_image_infer_surfaces_ripeness(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(
            FakeDetector(),
            model_version='1.0.0',
            schema_version='v1',
            ripeness_classifier=FakeRipenessClassifier(['harvestable']),
        )

        resp = client.post('/v1/recognition/image', files={'file': ('x.jpg', b'raw-bytes', 'image/jpeg')})
        assert resp.status_code == 200
        body = resp.json()
        assert body['result']['detections'][0]['ripeness'] == 'harvestable'


def test_health_and_stream_surface_model_load_error(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')
    load_error = 'No such file or directory: best.pt'

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(BrokenDetector(load_error), model_version='1.0.0', schema_version='v1')

        health = client.get('/v1/health')
        assert health.status_code == 200
        payload = health.json()
        assert payload['status'] == 'degraded'
        assert payload['model']['load_error'] == load_error

        resp = client.post('/v1/recognition/image', files={'file': ('x.jpg', b'raw-bytes', 'image/jpeg')})
        assert resp.status_code == 503
        assert load_error in resp.json()['detail']

        with client.websocket_connect('/v1/recognition/stream') as ws:
            ws.send_bytes(b'frame-bytes')
            msg = ws.receive_json()
            assert msg['type'] == 'error'
            assert load_error in msg['detail']


def test_health_surfaces_ripeness_load_error_without_breaking_detection(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(
            FakeDetector(),
            model_version='1.0.0',
            schema_version='v1',
            ripeness_classifier=FakeRipenessClassifier([], load_error='missing CAMELLIA_VLM_API_KEY'),
        )

        health = client.get('/v1/health')
        assert health.status_code == 200
        payload = health.json()
        assert payload['status'] == 'degraded'
        assert payload['model']['ripeness_enabled'] is True
        assert payload['model']['ripeness_loaded'] is False
        assert payload['model']['ripeness_error'] == 'missing CAMELLIA_VLM_API_KEY'

        resp = client.post('/v1/recognition/image', files={'file': ('x.jpg', b'raw-bytes', 'image/jpeg')})
        assert resp.status_code == 200
        assert resp.json()['result']['detections'][0]['ripeness'] is None


def test_stream_reuses_cached_ripeness_for_same_track(monkeypatch) -> None:
    recognition_router = importlib.import_module('api.recognition.router')

    monkeypatch.setenv('CAMELLIA_RECOGNITION_CONFIG', 'tooling/config/recognition.yaml.example')
    monkeypatch.setenv('CAMELLIA_SERVICE_CONFIG', 'tooling/config/service.yaml.example')
    monkeypatch.setattr(recognition_router, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(
            FakeDetector(),
            model_version='1.0.0',
            schema_version='v1',
            ripeness_classifier=FakeRipenessClassifier(['harvestable']),
        )

        with client.websocket_connect('/v1/recognition/stream') as ws:
            ws.send_bytes(b'frame-bytes')
            first = ws.receive_json()
            assert first['type'] == 'frame'
            assert first['result']['detections'][0]['ripeness'] is None

            time.sleep(0.05)

            ws.send_bytes(b'frame-bytes')
            second = ws.receive_json()
            assert second['type'] == 'frame'
            assert second['result']['detections'][0]['track_id'] == 1
            assert second['result']['detections'][0]['ripeness'] == 'harvestable'

            ws.send_text('eos')
            summary = ws.receive_json()
            assert summary['type'] == 'summary'
