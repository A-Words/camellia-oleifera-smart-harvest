package operations

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

func newOperationsHandler(t *testing.T) *Handler {
	t.Helper()

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
	t.Cleanup(func() { _ = db.Close() })

	return NewHandler(NewRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPlotAndTreeHandlers(t *testing.T) {
	handler := newOperationsHandler(t)
	mux := http.NewServeMux()
	handler.Register(mux)

	plotBody := `{"name":"试验地块","code":"plot-beta","row_count":4,"notes":"north ridge"}`
	plotReq := httptest.NewRequest(http.MethodPost, "/v1/operations/plots", bytes.NewBufferString(plotBody))
	plotRec := httptest.NewRecorder()
	mux.ServeHTTP(plotRec, plotReq)
	if plotRec.Code != http.StatusOK {
		t.Fatalf("create plot status = %d, want 200", plotRec.Code)
	}

	var plot Plot
	if err := json.Unmarshal(plotRec.Body.Bytes(), &plot); err != nil {
		t.Fatalf("decode plot: %v", err)
	}

	treeBody, _ := json.Marshal(map[string]any{
		"plot_id":   plot.PlotID,
		"tree_code": "B-01",
		"row_index": 1,
		"col_index": 1,
		"x":         10,
		"y":         20,
		"status":    "active",
	})
	treeReq := httptest.NewRequest(http.MethodPost, "/v1/operations/trees", bytes.NewReader(treeBody))
	treeRec := httptest.NewRecorder()
	mux.ServeHTTP(treeRec, treeReq)
	if treeRec.Code != http.StatusOK {
		t.Fatalf("create tree status = %d, want 200", treeRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/operations/plots/"+plot.PlotID+"/trees", nil)
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list trees status = %d, want 200", listRec.Code)
	}

	var response TreeListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode tree list: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("tree list len = %d, want 1", len(response.Items))
	}
}
