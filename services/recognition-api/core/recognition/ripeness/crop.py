from __future__ import annotations

from dataclasses import dataclass
from math import ceil, floor
from typing import Sequence

import numpy as np

from schemas.common import Detection


@dataclass(slots=True)
class DetectionCrop:
    detection_index: int
    track_id: int | None
    crop: np.ndarray


def crop_detection_regions(
    frame: np.ndarray,
    detections: Sequence[Detection],
    padding_ratio: float,
) -> list[DetectionCrop]:
    crops: list[DetectionCrop] = []
    if frame.ndim != 3:
        return crops

    height, width = frame.shape[:2]
    if width <= 0 or height <= 0:
        return crops

    for index, detection in enumerate(detections):
        crop = crop_detection_region(frame, detection.bbox, padding_ratio)
        if crop is None:
            continue
        crops.append(
            DetectionCrop(
                detection_index=index,
                track_id=detection.track_id,
                crop=crop,
            )
        )

    return crops


def crop_detection_region(
    frame: np.ndarray,
    bbox: tuple[float, float, float, float],
    padding_ratio: float,
) -> np.ndarray | None:
    if frame.ndim != 3:
        return None

    frame_height, frame_width = frame.shape[:2]
    if frame_width <= 0 or frame_height <= 0:
        return None

    x1, y1, x2, y2 = bbox
    box_width = max(0.0, x2 - x1)
    box_height = max(0.0, y2 - y1)
    if box_width <= 0 or box_height <= 0:
        return None

    pad_x = box_width * padding_ratio
    pad_y = box_height * padding_ratio

    crop_x1 = max(0, floor(x1 - pad_x))
    crop_y1 = max(0, floor(y1 - pad_y))
    crop_x2 = min(frame_width, ceil(x2 + pad_x))
    crop_y2 = min(frame_height, ceil(y2 + pad_y))
    if crop_x2 <= crop_x1 or crop_y2 <= crop_y1:
        return None

    crop = frame[crop_y1:crop_y2, crop_x1:crop_x2]
    if crop.size == 0:
        return None

    return crop
