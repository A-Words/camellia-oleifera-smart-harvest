from __future__ import annotations

import numpy as np

from core.recognition.ripeness.crop import crop_detection_region, crop_detection_regions
from schemas.common import Detection


def test_crop_detection_region_applies_padding_within_bounds() -> None:
    frame = np.zeros((100, 100, 3), dtype=np.uint8)

    crop = crop_detection_region(frame, (10, 20, 30, 40), padding_ratio=0.1)

    assert crop is not None
    assert crop.shape[:2] == (24, 24)


def test_crop_detection_region_clips_to_frame_bounds() -> None:
    frame = np.zeros((60, 80, 3), dtype=np.uint8)

    crop = crop_detection_region(frame, (0, 0, 20, 20), padding_ratio=0.2)

    assert crop is not None
    assert crop.shape[:2] == (24, 24)


def test_crop_detection_regions_skips_invalid_or_zero_area_boxes() -> None:
    frame = np.zeros((60, 80, 3), dtype=np.uint8)
    detections = [
        Detection(bbox=(10, 10, 30, 30), class_name='camellia_oleifera_fruit', confidence=0.9, track_id=1, ripeness=None),
        Detection(bbox=(10, 10, 10, 30), class_name='camellia_oleifera_fruit', confidence=0.9, track_id=2, ripeness=None),
    ]

    crops = crop_detection_regions(frame, detections, padding_ratio=0.12)

    assert len(crops) == 1
    assert crops[0].track_id == 1
