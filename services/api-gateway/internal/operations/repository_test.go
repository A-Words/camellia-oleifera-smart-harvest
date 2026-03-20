package operations

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/decision"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
)

func TestCreatePlotTreeAndWorkOrders(t *testing.T) {
	db, err := store.Open(config.DBConfig{
		Driver:           "sqlite",
		DSN:              filepath.Join(t.TempDir(), "operations.db"),
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		ConnMaxLifetimeS: 60,
	})
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	decisionRepo := decision.NewRepository(db)

	plot, err := repo.CreatePlot(context.Background(), CreatePlotRequest{
		Name:     "试验地块",
		Code:     "plot-alpha",
		RowCount: 3,
	})
	if err != nil {
		t.Fatalf("CreatePlot failed: %v", err)
	}

	tree, err := repo.CreateTree(context.Background(), CreateTreeRequest{
		PlotID:   plot.PlotID,
		TreeCode: "A-01",
		RowIndex: 1,
		ColIndex: 1,
		X:        1,
		Y:        1,
		Status:   TreeStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	plan := decision.DecisionPlan{
		PlanID:         "plan_1",
		PlotID:         plot.PlotID,
		GeneratedAt:    "2026-03-20T10:00:00Z",
		ManualOverride: false,
		Summary: decision.PlanSummary{
			TotalTrees:            1,
			ReadyTrees:            1,
			TotalHarvestableCount: 1,
		},
		TreeSequence: []decision.PlanTreeSequenceItem{
			{TreeID: tree.TreeID, TreeCode: tree.TreeCode, PriorityOrder: 1, Status: decision.TreeRecommendationReady},
		},
		TreeRecommendations: []decision.TreeRecommendation{
			{
				TreeID:   tree.TreeID,
				TreeCode: tree.TreeCode,
				Status:   decision.TreeRecommendationReady,
				ZonePriorities: []decision.ZonePriority{
					{Order: 1, Zone: "lower_center"},
				},
				PickSequence: []decision.PickSequenceItem{
					{Order: 1, DetectionIndex: 0, Zone: "lower_center", Confidence: 0.9},
				},
				SkipItems: []decision.SkipItem{},
			},
		},
	}
	if err := decisionRepo.SavePlan(context.Background(), plan); err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}

	workOrders, err := repo.CreateWorkOrdersFromPlan(context.Background(), plan.PlanID)
	if err != nil {
		t.Fatalf("CreateWorkOrdersFromPlan failed: %v", err)
	}
	if len(workOrders) != 1 {
		t.Fatalf("work_orders len = %d, want 1", len(workOrders))
	}
	if workOrders[0].Status != WorkOrderStatusPending {
		t.Fatalf("work order status = %s, want pending", workOrders[0].Status)
	}
}
