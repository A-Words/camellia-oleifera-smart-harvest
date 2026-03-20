from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Literal, Sequence

import numpy as np

RipenessLabel = Literal["harvestable", "not_ready", "occluded_unclear"]


class RipenessClassifier(ABC):
    name: str = "base"
    load_error: str | None = None
    crop_padding_ratio: float = 0.12

    @abstractmethod
    def load(self) -> None:
        raise NotImplementedError

    @property
    @abstractmethod
    def loaded(self) -> bool:
        raise NotImplementedError

    @abstractmethod
    async def classify_crops(self, crops: Sequence[np.ndarray]) -> list[RipenessLabel | None]:
        raise NotImplementedError


@dataclass(slots=True)
class BrokenRipenessClassifier(RipenessClassifier):
    name: str
    load_error: str
    crop_padding_ratio: float = 0.12

    @property
    def loaded(self) -> bool:
        return False

    def load(self) -> None:
        return

    async def classify_crops(self, crops: Sequence[np.ndarray]) -> list[RipenessLabel | None]:
        return [None] * len(crops)
