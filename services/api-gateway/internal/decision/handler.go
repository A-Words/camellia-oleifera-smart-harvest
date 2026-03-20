package decision

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
)

const historyLimit = 10

type Handler struct {
	engine     *Engine
	repository *Repository
	logger     *slog.Logger
}

func NewHandler(repository *Repository, logger *slog.Logger) *Handler {
	return &Handler{
		engine:     NewEngine(),
		repository: repository,
		logger:     logger,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/decision/recommendation", h.handleRecommendation)
	mux.HandleFunc("GET /v1/decision/history", h.handleHistory)
	mux.HandleFunc("POST /v1/decision/observations", h.handleCreateObservation)
	mux.HandleFunc("GET /v1/decision/observations", h.handleListObservations)
	mux.HandleFunc("POST /v1/decision/plans", h.handleCreatePlan)
	mux.HandleFunc("GET /v1/decision/plans", h.handleListPlans)
	mux.HandleFunc("PATCH /v1/decision/plans/{plan_id}", h.handleUpdatePlan)
}

func (h *Handler) handleRecommendation(w http.ResponseWriter, r *http.Request) {
	var request SnapshotRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	decisionID, err := store.NewID("decision")
	if err != nil {
		h.logger.Error("decision id generation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "生成决策编号失败")
		return
	}

	response, err := h.engine.Recommend(decisionID, now, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.repository.SaveTreeHistory(r.Context(), request, response); err != nil {
		h.logger.Error("decision save failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "保存决策历史失败")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleHistory(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.ListRecentTreeHistory(r.Context(), historyLimit)
	if err != nil {
		h.logger.Error("decision history query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取决策历史失败")
		return
	}

	writeJSON(w, http.StatusOK, HistoryResponse{Items: items})
}

func (h *Handler) handleCreateObservation(w http.ResponseWriter, r *http.Request) {
	var request ObservationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(request.TreeID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "tree_id 不能为空")
		return
	}
	if request.FrameWidth <= 0 || request.FrameHeight <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "frame_width 和 frame_height 必须大于 0")
		return
	}

	observationID, err := store.NewID("observation")
	if err != nil {
		h.logger.Error("observation id generation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "生成观测编号失败")
		return
	}

	capturedAt := request.CapturedAt
	if strings.TrimSpace(capturedAt) == "" {
		capturedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	observation := Observation{
		ObservationID: observationID,
		TreeID:        request.TreeID,
		CapturedAt:    capturedAt,
		FrameIndex:    request.FrameIndex,
		TimestampMS:   request.TimestampMS,
		FrameWidth:    request.FrameWidth,
		FrameHeight:   request.FrameHeight,
		Detections:    request.Detections,
	}

	if _, err := h.engine.analyzeTree(request.TreeID, "", nil, SnapshotRequest{
		FrameIndex:  request.FrameIndex,
		TimestampMS: request.TimestampMS,
		FrameWidth:  request.FrameWidth,
		FrameHeight: request.FrameHeight,
		Detections:  request.Detections,
	}); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.repository.CreateObservation(r.Context(), observation); err != nil {
		h.logger.Error("create observation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "保存树观测失败")
		return
	}

	writeJSON(w, http.StatusOK, observation)
}

func (h *Handler) handleListObservations(w http.ResponseWriter, r *http.Request) {
	treeID := strings.TrimSpace(r.URL.Query().Get("tree_id"))
	if treeID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "tree_id 不能为空")
		return
	}

	items, err := h.repository.ListObservationsByTree(r.Context(), treeID)
	if err != nil {
		h.logger.Error("list observations failed", "tree_id", treeID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取树观测失败")
		return
	}
	writeJSON(w, http.StatusOK, ObservationListResponse{Items: items})
}

func (h *Handler) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var request PlanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(request.PlotID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plot_id 不能为空")
		return
	}

	inputs, err := h.repository.LoadLatestObservationsForPlot(r.Context(), request.PlotID)
	if err != nil {
		h.logger.Error("load latest observations failed", "plot_id", request.PlotID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取地块树观测失败")
		return
	}

	planID, err := store.NewID("plan")
	if err != nil {
		h.logger.Error("plan id generation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "生成计划编号失败")
		return
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339Nano)
	plan, err := h.engine.BuildPlan(planID, generatedAt, request.PlotID, inputs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.repository.SavePlan(r.Context(), plan); err != nil {
		h.logger.Error("save plan failed", "plot_id", request.PlotID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "保存决策计划失败")
		return
	}
	if err := h.repository.SavePlanAudit(r.Context(), plan); err != nil {
		h.logger.Warn("save plan audit failed", "plan_id", plan.PlanID, "error", err)
	}

	writeJSON(w, http.StatusOK, plan)
}

func (h *Handler) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plotID := strings.TrimSpace(r.URL.Query().Get("plot_id"))
	if plotID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plot_id 不能为空")
		return
	}

	items, err := h.repository.ListPlansByPlot(r.Context(), plotID)
	if err != nil {
		h.logger.Error("list plans failed", "plot_id", plotID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取决策计划失败")
		return
	}
	writeJSON(w, http.StatusOK, DecisionPlanListResponse{Items: items})
}

func (h *Handler) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("plan_id")
	if strings.TrimSpace(planID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "plan_id 不能为空")
		return
	}

	var request UpdatePlanRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	plan, err := h.repository.LoadPlan(r.Context(), planID)
	if err != nil {
		h.logger.Error("load plan failed", "plan_id", planID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	trees, err := h.repository.LoadTreeMetaForPlot(r.Context(), plan.PlotID)
	if err != nil {
		h.logger.Error("load tree meta failed", "plan_id", planID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取树坐标失败")
		return
	}

	nextPlan, err := h.engine.ApplyManualOverride(plan, trees, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	nextPlan.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)

	if err := h.repository.UpdatePlan(r.Context(), nextPlan); err != nil {
		h.logger.Error("update plan failed", "plan_id", planID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "保存人工调整失败")
		return
	}

	writeJSON(w, http.StatusOK, nextPlan)
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
