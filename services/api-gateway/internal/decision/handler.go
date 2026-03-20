package decision

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
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
}

func (h *Handler) handleRecommendation(w http.ResponseWriter, r *http.Request) {
	var request SnapshotRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	decisionID, err := newDecisionID()
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

	if err := h.repository.Save(r.Context(), request, response); err != nil {
		h.logger.Error("decision save failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "保存决策历史失败")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleHistory(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.ListRecent(r.Context(), historyLimit)
	if err != nil {
		h.logger.Error("decision history query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "读取决策历史失败")
		return
	}

	writeJSON(w, http.StatusOK, HistoryResponse{Items: items})
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

func newDecisionID() (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "decision_" + hex.EncodeToString(randomBytes), nil
}
