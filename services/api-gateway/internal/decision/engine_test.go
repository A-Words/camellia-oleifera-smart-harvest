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

func TestBuildPlanOrdersTreesByHarvestableSignalAndDistance(t *testing.T) {
	engine := NewEngine()
	harvestable := RipenessHarvestable

	plan, err := engine.BuildPlan("plan_1", "2026-03-20T10:00:00Z", "plot_1", []latestObservation{
		{
			Tree: TreeMeta{TreeID: "tree_a", PlotID: "plot_1", TreeCode: "A-01", X: 0, Y: 0, Status: "active"},
			Observation: &Observation{
				ObservationID: "obs_a",
				TreeID:        "tree_a",
				CapturedAt:    "2026-03-20T10:00:00Z",
				FrameWidth:    300,
				FrameHeight:   300,
				Detections: []Detection{
					{BBox: [4]float64{10, 210, 70, 270}, ClassName: "camellia_oleifera_fruit", Confidence: 0.95, Ripeness: &harvestable},
				},
			},
		},
		{
			Tree: TreeMeta{TreeID: "tree_b", PlotID: "plot_1", TreeCode: "A-02", X: 1, Y: 0, Status: "active"},
			Observation: &Observation{
				ObservationID: "obs_b",
				TreeID:        "tree_b",
				CapturedAt:    "2026-03-20T10:00:00Z",
				FrameWidth:    300,
				FrameHeight:   300,
				Detections: []Detection{
					{BBox: [4]float64{10, 210, 70, 270}, ClassName: "camellia_oleifera_fruit", Confidence: 0.95, Ripeness: &harvestable},
					{BBox: [4]float64{120, 190, 180, 250}, ClassName: "camellia_oleifera_fruit", Confidence: 0.88, Ripeness: &harvestable},
				},
			},
		},
		{
			Tree:        TreeMeta{TreeID: "tree_c", PlotID: "plot_1", TreeCode: "A-03", X: 2, Y: 0, Status: "active"},
			Observation: nil,
		},
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	if plan.Summary.TotalTrees != 3 {
		t.Fatalf("total_trees = %d, want 3", plan.Summary.TotalTrees)
	}
	if plan.Summary.ReadyTrees != 2 {
		t.Fatalf("ready_trees = %d, want 2", plan.Summary.ReadyTrees)
	}
	if plan.Summary.PendingObservationTrees != 1 {
		t.Fatalf("pending_observation_trees = %d, want 1", plan.Summary.PendingObservationTrees)
	}
	if len(plan.TreeSequence) != 2 {
		t.Fatalf("tree sequence len = %d, want 2", len(plan.TreeSequence))
	}
	if plan.TreeSequence[0].TreeID != "tree_b" {
		t.Fatalf("first tree = %s, want tree_b", plan.TreeSequence[0].TreeID)
	}
}

func TestApplyManualOverrideReordersSequenceAndAddsManualSkip(t *testing.T) {
	engine := NewEngine()
	mainZone := "lower_center"
	observationID := "obs_1"

	plan, err := engine.ApplyManualOverride(DecisionPlan{
		PlanID:         "plan_1",
		PlotID:         "plot_1",
		GeneratedAt:    "2026-03-20T10:00:00Z",
		ManualOverride: false,
		TreeSequence: []PlanTreeSequenceItem{
			{TreeID: "tree_a", TreeCode: "A-01", PriorityOrder: 1},
			{TreeID: "tree_b", TreeCode: "A-02", PriorityOrder: 2},
		},
		TreeRecommendations: []TreeRecommendation{
			{
				TreeID:        "tree_a",
				TreeCode:      "A-01",
				ObservationID: &observationID,
				Status:        TreeRecommendationReady,
				Summary: Summary{
					HarvestableCount: 2,
					SkippedCount:     0,
					MainPriorityZone: &mainZone,
				},
				ZonePriorities: []ZonePriority{
					{Order: 1, Zone: "lower_center"},
					{Order: 2, Zone: "lower_left"},
				},
				PickSequence: []PickSequenceItem{
					{Order: 1, DetectionIndex: 10, Zone: "lower_center", Confidence: 0.9},
					{Order: 2, DetectionIndex: 11, Zone: "lower_left", Confidence: 0.8},
				},
				SkipItems: []SkipItem{},
			},
		},
		Summary: PlanSummary{TotalTrees: 2, ReadyTrees: 1, TotalHarvestableCount: 2},
	}, map[string]TreeMeta{
		"tree_a": {TreeID: "tree_a", X: 0, Y: 0},
		"tree_b": {TreeID: "tree_b", X: 1, Y: 0},
	}, UpdatePlanRequest{
		TreeSequence: []string{"tree_b", "tree_a"},
		TreeOverrides: []TreeOverride{
			{
				TreeID:                        "tree_a",
				ZoneOrder:                     []string{"lower_left", "lower_center"},
				PickSequenceDetectionIndices:  []int{11, 10},
				ManualSkippedDetectionIndices: []int{10},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyManualOverride failed: %v", err)
	}

	if !plan.ManualOverride {
		t.Fatal("manual_override should be true")
	}
	if plan.TreeSequence[0].TreeID != "tree_b" {
		t.Fatalf("first tree after override = %s, want tree_b", plan.TreeSequence[0].TreeID)
	}
	if plan.TreeRecommendations[0].ZonePriorities[0].Zone != "lower_left" {
		t.Fatalf("first zone after override = %s, want lower_left", plan.TreeRecommendations[0].ZonePriorities[0].Zone)
	}
	if len(plan.TreeRecommendations[0].PickSequence) != 1 || plan.TreeRecommendations[0].PickSequence[0].DetectionIndex != 11 {
		t.Fatalf("pick_sequence after override = %+v", plan.TreeRecommendations[0].PickSequence)
	}
	if len(plan.TreeRecommendations[0].SkipItems) != 1 || plan.TreeRecommendations[0].SkipItems[0].SkipReason != SkipReasonManual {
		t.Fatalf("skip_items after override = %+v", plan.TreeRecommendations[0].SkipItems)
	}
}
