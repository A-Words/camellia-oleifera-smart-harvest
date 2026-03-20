from __future__ import annotations

import asyncio
import base64
import json
import os
from typing import Any, Sequence

import httpx
import numpy as np

from core.recognition.ripeness.base import RipenessClassifier, RipenessLabel
from settings import ModelConfig

RIPENESS_PROMPT = (
    "你是油茶果成熟度判定器。"
    "请只根据给定裁剪图判断该目标是否 `harvestable`、`not_ready` 或 `occluded_unclear`。"
    "只返回 JSON，对象结构固定为 {\"ripeness\":\"harvestable|not_ready|occluded_unclear\"}。"
    "不要输出额外说明、Markdown 或代码块。"
)

_VALID_RIPENESS: set[str] = {"harvestable", "not_ready", "occluded_unclear"}


class OpenAICompatibleRipenessClassifier(RipenessClassifier):
    name = "openai_compatible"

    def __init__(self, cfg: ModelConfig) -> None:
        self.cfg = cfg
        self.crop_padding_ratio = cfg.vlm_crop_padding_ratio
        self.load_error: str | None = None
        self._loaded = False
        self._api_key = ""

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        if not self.cfg.vlm_model.strip():
            self._mark_load_failed("VLM model is not configured")
            return
        if not self.cfg.vlm_base_url.strip():
            self._mark_load_failed("VLM base URL is not configured")
            return

        api_key = os.getenv("CAMELLIA_VLM_API_KEY", "").strip()
        if not api_key:
            self._mark_load_failed("CAMELLIA_VLM_API_KEY is required when ripeness_enabled=true")
            return

        self._api_key = api_key
        self._loaded = True
        self.load_error = None

    async def classify_crops(self, crops: Sequence[np.ndarray]) -> list[RipenessLabel | None]:
        if not self.loaded:
            return [None] * len(crops)
        if not crops:
            return []

        results: list[RipenessLabel | None] = []
        async with httpx.AsyncClient(
            base_url=self.cfg.vlm_base_url.rstrip("/"),
            timeout=self.cfg.vlm_timeout_s,
        ) as client:
            for chunk in _chunked(list(crops), self.cfg.vlm_batch_size):
                chunk_results = await asyncio.gather(
                    *(self._classify_single_crop(client, crop) for crop in chunk),
                    return_exceptions=True,
                )
                for item in chunk_results:
                    if isinstance(item, Exception):
                        results.append(None)
                    else:
                        results.append(item)
        return results

    async def _classify_single_crop(
        self,
        client: httpx.AsyncClient,
        crop: np.ndarray,
    ) -> RipenessLabel | None:
        data_url = _encode_crop_as_data_url(crop)
        if not data_url:
            return None

        response = await client.post(
            "chat/completions",
            headers={
                "Authorization": f"Bearer {self._api_key}",
                "Content-Type": "application/json",
            },
            json={
                "model": self.cfg.vlm_model,
                "temperature": 0,
                "max_tokens": 32,
                "messages": [
                    {
                        "role": "system",
                        "content": RIPENESS_PROMPT,
                    },
                    {
                        "role": "user",
                        "content": [
                            {
                                "type": "text",
                                "text": "请只返回 JSON。",
                            },
                            {
                                "type": "image_url",
                                "image_url": {
                                    "url": data_url,
                                },
                            },
                        ],
                    },
                ],
            },
        )
        response.raise_for_status()
        return _parse_ripeness_label(response.json())

    def _mark_load_failed(self, reason: str) -> None:
        self._loaded = False
        self.load_error = reason


def _encode_crop_as_data_url(crop: np.ndarray) -> str | None:
    try:
        import cv2
    except Exception:
        return None

    ok, encoded = cv2.imencode(".jpg", crop)
    if not ok:
        return None
    payload = base64.b64encode(encoded.tobytes()).decode("ascii")
    return f"data:image/jpeg;base64,{payload}"


def _parse_ripeness_label(payload: dict[str, Any]) -> RipenessLabel | None:
    content = _extract_message_content(payload)
    if not content:
        return None

    try:
        data = json.loads(_strip_json_wrapper(content))
    except json.JSONDecodeError:
        return None

    if not isinstance(data, dict):
        return None

    ripeness = data.get("ripeness")
    if isinstance(ripeness, str) and ripeness in _VALID_RIPENESS:
        return ripeness  # type: ignore[return-value]
    return None


def _extract_message_content(payload: dict[str, Any]) -> str:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        return ""

    message = choices[0].get("message")
    if not isinstance(message, dict):
        return ""

    content = message.get("content")
    if isinstance(content, str):
        return content.strip()

    if isinstance(content, list):
        text_parts: list[str] = []
        for item in content:
            if not isinstance(item, dict):
                continue
            if item.get("type") == "text" and isinstance(item.get("text"), str):
                text_parts.append(item["text"])
        return "\n".join(text_parts).strip()

    return ""


def _strip_json_wrapper(content: str) -> str:
    stripped = content.strip()
    if stripped.startswith("```"):
        lines = [line for line in stripped.splitlines() if not line.startswith("```")]
        stripped = "\n".join(lines).strip()
    return stripped


def _chunked(items: list[np.ndarray], size: int) -> list[list[np.ndarray]]:
    return [items[index:index + size] for index in range(0, len(items), size)]
