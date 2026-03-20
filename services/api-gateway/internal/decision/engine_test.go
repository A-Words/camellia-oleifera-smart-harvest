package decision

import "testing"

func TestRecommendBuildsZonePriorityAndPickSequence(t *testing.T) {
	engine := NewEngine()
	harvestable := RipenessHarvestable
	notReady := RipenessNotReady

	resp, err := engine.Recommend("decision_1", "2026-03-20T10:00:00Z", SnapshotRequest{
		FrameIndex:  3,
		TimestampMS: 1000,
		FrameWidth:  300,
		FrameHeight: 300,
		Detections: []Detection{
			{BBox: [4]float64{10, 210, 70, 270}, ClassName: "camellia_oleifera_fruit", Confidence: 0.95, Ripeness: &harvestable},
			{BBox: [4]float64{120, 190, 180, 250}, ClassName: "camellia_oleifera_fruit", Confidence: 0.88, Ripeness: &harvestable},
			{BBox: [4]float64{220, 40, 280, 100}, ClassName: "camellia_oleifera_fruit", Confidence: 0.72, Ripeness: &notReady},
		},
	})
	if err != nil {
		t.Fatalf("Recommend failed: %v", err)
	}

	if resp.Summary.HarvestableCount != 2 {
		t.Fatalf("harvestable_count = %d, want 2", resp.Summary.HarvestableCount)
	}
	if resp.Summary.SkippedCount != 1 {
		t.Fatalf("skipped_count = %d, want 1", resp.Summary.SkippedCount)
	}
	if resp.Summary.MainPriorityZone == nil || *resp.Summary.MainPriorityZone != "lower_center" {
		t.Fatalf("main_priority_zone = %v, want lower_center", resp.Summary.MainPriorityZone)
	}
	if len(resp.ZonePriorities) != 2 {
		t.Fatalf("zone priorities len = %d, want 2", len(resp.ZonePriorities))
	}
	if resp.ZonePriorities[0].Zone != "lower_center" {
		t.Fatalf("first zone = %s, want lower_center", resp.ZonePriorities[0].Zone)
	}
	if len(resp.PickSequence) != 2 {
		t.Fatalf("pick sequence len = %d, want 2", len(resp.PickSequence))
	}
	if resp.PickSequence[0].DetectionIndex != 1 {
		t.Fatalf("first pick detection index = %d, want 1", resp.PickSequence[0].DetectionIndex)
	}
	if len(resp.SkipItems) != 1 || resp.SkipItems[0].SkipReason != RipenessNotReady {
		t.Fatalf("skip item = %+v, want not_ready", resp.SkipItems)
	}
}

func TestRecommendTreatsMissingRipenessAsOccludedSkip(t *testing.T) {
	engine := NewEngine()

	resp, err := engine.Recommend("decision_2", "2026-03-20T10:00:00Z", SnapshotRequest{
		FrameWidth:  300,
		FrameHeight: 300,
		Detections: []Detection{
			{BBox: [4]float64{50, 50, 80, 80}, ClassName: "camellia_oleifera_fruit", Confidence: 0.9, Ripeness: nil},
		},
	})
	if err != nil {
		t.Fatalf("Recommend failed: %v", err)
	}

	if len(resp.PickSequence) != 0 {
		t.Fatalf("pick sequence len = %d, want 0", len(resp.PickSequence))
	}
	if len(resp.SkipItems) != 1 {
		t.Fatalf("skip items len = %d, want 1", len(resp.SkipItems))
	}
	if resp.SkipItems[0].SkipReason != RipenessOccludedUnclear {
		t.Fatalf("skip reason = %s, want occluded_unclear", resp.SkipItems[0].SkipReason)
	}
}

func TestRecommendRejectsInvalidFrameDimensions(t *testing.T) {
	engine := NewEngine()

	_, err := engine.Recommend("decision_3", "2026-03-20T10:00:00Z", SnapshotRequest{
		FrameWidth:  0,
		FrameHeight: 200,
	})
	if err == nil {
		t.Fatal("expected error for invalid frame dimensions")
	}
}
