from core.recognition.aggregator import SessionAggregator


def test_summary_reports_total_detected() -> None:
    agg = SessionAggregator()
    agg.update_session(list(range(10)))
    summary = agg.build_summary()
    assert summary.total_detected == 10


def test_frame_summary_uses_detection_count() -> None:
    agg = SessionAggregator()
    summary = agg.frame_summary(3)
    assert summary.total == 3


def test_deduplicate_by_track_id() -> None:
    agg = SessionAggregator()
    agg.update_session([1])
    agg.update_session([1])
    summary = agg.build_summary()
    assert summary.total_detected == 1
