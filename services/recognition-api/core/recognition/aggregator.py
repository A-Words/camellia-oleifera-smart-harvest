from __future__ import annotations

from dataclasses import dataclass, field

from schemas.common import FrameSummary, SessionSummary


@dataclass
class SessionAggregator:
    seen_track_ids: set[int] = field(default_factory=set)
    total_unique: int = 0

    def frame_summary(self, detection_count: int) -> FrameSummary:
        return FrameSummary(total=detection_count)

    def update_session(self, track_ids: list[int | None]) -> None:
        for track_id in track_ids:
            if track_id is not None:
                if track_id in self.seen_track_ids:
                    continue
                self.seen_track_ids.add(track_id)
            self.total_unique += 1

    def build_summary(self) -> SessionSummary:
        return SessionSummary(total_detected=self.total_unique)
