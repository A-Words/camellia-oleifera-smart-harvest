package decision

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
)

func TestRepositorySaveAndListRecent(t *testing.T) {
	repo, err := NewRepository(config.DBConfig{
		Driver:           "sqlite",
		DSN:              filepath.Join(t.TempDir(), "decision.db"),
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		ConnMaxLifetimeS: 60,
	})
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}
	defer repo.Close()

	harvestable := RipenessHarvestable
	request := SnapshotRequest{
		FrameWidth:  100,
		FrameHeight: 100,
		Detections: []Detection{
			{BBox: [4]float64{10, 10, 30, 30}, ClassName: "camellia_oleifera_fruit", Confidence: 0.9, Ripeness: &harvestable},
		},
	}
	response := RecommendationResponse{
		DecisionID: "decision_a",
		CreatedAt:  "2026-03-20T10:00:00Z",
		Summary: Summary{
			TotalDetections:  1,
			HarvestableCount: 1,
		},
	}

	if err := repo.Save(context.Background(), request, response); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	items, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("history len = %d, want 1", len(items))
	}
	if items[0].DecisionID != "decision_a" {
		t.Fatalf("decision_id = %s, want decision_a", items[0].DecisionID)
	}
	if len(items[0].Request.Detections) != 1 {
		t.Fatalf("detections len = %d, want 1", len(items[0].Request.Detections))
	}
}
