package decision

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	db, err := store.Open(config.DBConfig{
		Driver:           "sqlite",
		DSN:              filepath.Join(t.TempDir(), "decision.db"),
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		ConnMaxLifetimeS: 60,
	})
	if err != nil {
		t.Fatalf("store.Open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return NewHandler(NewRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRecommendationHandlerSuccess(t *testing.T) {
	handler := newTestHandler(t)
	mux := http.NewServeMux()
	handler.Register(mux)

	payload := map[string]any{
		"frame_index":  1,
		"timestamp_ms": 1000,
		"frame_width":  300,
		"frame_height": 300,
		"detections": []map[string]any{
			{
				"bbox":       []float64{100, 200, 160, 260},
				"class_name": "camellia_oleifera_fruit",
				"confidence": 0.9,
				"ripeness":   "harvestable",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/decision/recommendation", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response RecommendationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Summary.HarvestableCount != 1 {
		t.Fatalf("harvestable_count = %d, want 1", response.Summary.HarvestableCount)
	}
}

func TestRecommendationHandlerAllowsEmptyDetections(t *testing.T) {
	handler := newTestHandler(t)
	mux := http.NewServeMux()
	handler.Register(mux)

	payload := `{"frame_index":1,"timestamp_ms":1000,"frame_width":300,"frame_height":300,"detections":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/decision/recommendation", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response RecommendationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.PickSequence) != 0 || len(response.SkipItems) != 0 {
		t.Fatalf("unexpected non-empty response: %+v", response)
	}
}

func TestRecommendationHandlerTreatsMissingRipenessAsSkip(t *testing.T) {
	handler := newTestHandler(t)
	mux := http.NewServeMux()
	handler.Register(mux)

	payload := `{"frame_index":1,"timestamp_ms":1000,"frame_width":300,"frame_height":300,"detections":[{"bbox":[1,2,10,11],"class_name":"camellia_oleifera_fruit","confidence":0.6}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/decision/recommendation", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response RecommendationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.SkipItems) != 1 || response.SkipItems[0].SkipReason != RipenessOccludedUnclear {
		t.Fatalf("skip items = %+v", response.SkipItems)
	}
}

func TestHistoryHandlerReturnsRecentItems(t *testing.T) {
	handler := newTestHandler(t)
	mux := http.NewServeMux()
	handler.Register(mux)

	for _, payload := range []string{
		`{"frame_index":1,"timestamp_ms":1000,"frame_width":300,"frame_height":300,"detections":[]}`,
		`{"frame_index":2,"timestamp_ms":2000,"frame_width":300,"frame_height":300,"detections":[]}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/decision/recommendation", bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("recommendation status = %d, want 200", rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/decision/history", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response HistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("history len = %d, want 2", len(response.Items))
	}
	if response.Items[0].Request.FrameIndex != 2 {
		t.Fatalf("first history frame_index = %d, want 2", response.Items[0].Request.FrameIndex)
	}
}
