package operations

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type Handler struct {
	repository *Repository
	logger     *slog.Logger
}

func NewHandler(repository *Repository, logger *slog.Logger) *Handler {
	return &Handler{
		repository: repository,
		logger:     logger,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/operations/plots", h.handleCreatePlot)
	mux.HandleFunc("GET /v1/operations/plots", h.handleListPlots)
	mux.HandleFunc("POST /v1/operations/trees", h.handleCreateTree)
	mux.HandleFunc("GET /v1/operations/plots/{plot_id}/trees", h.handleListTreesByPlot)
	mux.HandleFunc("PATCH /v1/operations/trees/{tree_id}", h.handleUpdateTree)
	mux.HandleFunc("POST /v1/operations/work-orders", h.handleCreateWorkOrders)
	mux.HandleFunc("GET /v1/operations/work-orders", h.handleListWorkOrders)
	mux.HandleFunc("PATCH /v1/operations/work-orders/{work_order_id}", h.handleUpdateWorkOrder)
}

func (h *Handler) handleCreatePlot(w http.ResponseWriter, r *http.Request) {
	var request CreatePlotRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Code) == "" || request.RowCount <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "name、code、row_count 为必填项")
		return
	}

	plot, err := h.repository.CreatePlot(r.Context(), request)
	if err != nil {
		h.logger.Error("create plot failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "创建地块失败")
		return
	}
	writeJSON(w, http.StatusOK, plot)
}

func (h *Handler) handleListPlots(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.ListPlots(r.Context())
	if err != nil {
		h.logger.Error("list plots failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取地块失败")
		return
	}
	writeJSON(w, http.StatusOK, PlotListResponse{Items: items})
}

func (h *Handler) handleCreateTree(w http.ResponseWriter, r *http.Request) {
	var request CreateTreeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(request.PlotID) == "" || strings.TrimSpace(request.TreeCode) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plot_id、tree_code 为必填项")
		return
	}
	if request.Status == "" {
		request.Status = TreeStatusActive
	}
	if request.Status != TreeStatusActive && request.Status != TreeStatusDisabled {
		writeError(w, http.StatusBadRequest, "bad_request", "status 仅支持 active 或 disabled")
		return
	}

	tree, err := h.repository.CreateTree(r.Context(), request)
	if err != nil {
		h.logger.Error("create tree failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "创建树档案失败")
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) handleListTreesByPlot(w http.ResponseWriter, r *http.Request) {
	plotID := r.PathValue("plot_id")
	if plotID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plot_id 不能为空")
		return
	}

	items, err := h.repository.ListTreesByPlot(r.Context(), plotID)
	if err != nil {
		h.logger.Error("list trees failed", "plot_id", plotID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取树档案失败")
		return
	}
	writeJSON(w, http.StatusOK, TreeListResponse{Items: items})
}

func (h *Handler) handleUpdateTree(w http.ResponseWriter, r *http.Request) {
	treeID := r.PathValue("tree_id")
	if treeID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "tree_id 不能为空")
		return
	}

	var request UpdateTreeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if request.Status != nil && *request.Status != TreeStatusActive && *request.Status != TreeStatusDisabled {
		writeError(w, http.StatusBadRequest, "bad_request", "status 仅支持 active 或 disabled")
		return
	}

	tree, err := h.repository.UpdateTree(r.Context(), treeID, request)
	if err != nil {
		h.logger.Error("update tree failed", "tree_id", treeID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) handleCreateWorkOrders(w http.ResponseWriter, r *http.Request) {
	var request CreateWorkOrdersRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(request.PlanID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plan_id 不能为空")
		return
	}

	items, err := h.repository.CreateWorkOrdersFromPlan(r.Context(), request.PlanID)
	if err != nil {
		h.logger.Error("create work orders failed", "plan_id", request.PlanID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "创建作业单失败")
		return
	}
	writeJSON(w, http.StatusOK, WorkOrderListResponse{Items: items})
}

func (h *Handler) handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	plotID := strings.TrimSpace(r.URL.Query().Get("plot_id"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !isValidWorkOrderStatus(status) {
		writeError(w, http.StatusBadRequest, "bad_request", "status 不合法")
		return
	}

	items, err := h.repository.ListWorkOrders(r.Context(), plotID, status)
	if err != nil {
		h.logger.Error("list work orders failed", "plot_id", plotID, "status", status, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取作业单失败")
		return
	}
	writeJSON(w, http.StatusOK, WorkOrderListResponse{Items: items})
}

func (h *Handler) handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request) {
	workOrderID := r.PathValue("work_order_id")
	if workOrderID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "work_order_id 不能为空")
		return
	}

	var request UpdateWorkOrderRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if !isValidWorkOrderStatus(request.Status) {
		writeError(w, http.StatusBadRequest, "bad_request", "status 不合法")
		return
	}
	if request.Status == WorkOrderStatusSkipped && (request.SkipReasonNote == nil || strings.TrimSpace(*request.SkipReasonNote) == "") {
		writeError(w, http.StatusBadRequest, "bad_request", "skipped 状态必须提供 skip_reason_note")
		return
	}

	workOrder, err := h.repository.UpdateWorkOrder(r.Context(), workOrderID, request)
	if err != nil {
		h.logger.Error("update work order failed", "work_order_id", workOrderID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workOrder)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(w, statusCode, map[string]string{
		"error":   code,
		"message": message,
	})
}

func isValidWorkOrderStatus(status string) bool {
	switch status {
	case WorkOrderStatusPending, WorkOrderStatusInProgress, WorkOrderStatusCompleted, WorkOrderStatusSkipped:
		return true
	default:
		return false
	}
}
