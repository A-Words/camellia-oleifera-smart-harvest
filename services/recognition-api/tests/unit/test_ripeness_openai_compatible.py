from __future__ import annotations

from core.recognition.ripeness.openai_compatible import _parse_ripeness_label
from settings import ModelConfig
from core.recognition.ripeness.openai_compatible import OpenAICompatibleRipenessClassifier


def test_parse_ripeness_label_accepts_json_payload() -> None:
    payload = {
        "choices": [
            {
                "message": {
                    "content": "{\"ripeness\":\"harvestable\"}",
                }
            }
        ]
    }

    assert _parse_ripeness_label(payload) == "harvestable"


def test_parse_ripeness_label_rejects_invalid_value() -> None:
    payload = {
        "choices": [
            {
                "message": {
                    "content": "```json\n{\"ripeness\":\"unknown\"}\n```",
                }
            }
        ]
    }

    assert _parse_ripeness_label(payload) is None


def test_openai_compatible_classifier_requires_api_key(monkeypatch) -> None:
    monkeypatch.delenv("CAMELLIA_VLM_API_KEY", raising=False)
    classifier = OpenAICompatibleRipenessClassifier(
        ModelConfig(
            ripeness_enabled=True,
            vlm_model="gpt-4.1-mini",
            vlm_base_url="https://example.com/v1",
        )
    )

    classifier.load()

    assert classifier.loaded is False
    assert classifier.load_error == "CAMELLIA_VLM_API_KEY is required when ripeness_enabled=true"
