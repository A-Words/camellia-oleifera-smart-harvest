package operations

import "github.com/camellia-oleifera-smart-harvest/api-gateway/internal/decision"

const (
	TreeStatusActive   = "active"
	TreeStatusDisabled = "disabled"

	WorkOrderStatusPending    = "pending"
	WorkOrderStatusInProgress = "in_progress"
	WorkOrderStatusCompleted  = "completed"
	WorkOrderStatusSkipped    = "skipped"
)

type Plot struct {
	PlotID    string `json:"plot_id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	RowCount  int    `json:"row_count"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreatePlotRequest struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	RowCount int    `json:"row_count"`
	Notes    string `json:"notes"`
}

type PlotListResponse struct {
	Items []Plot `json:"items"`
}

type TreeArchive struct {
	TreeID    string  `json:"tree_id"`
	PlotID    string  `json:"plot_id"`
	TreeCode  string  `json:"tree_code"`
	RowIndex  int     `json:"row_index"`
	ColIndex  int     `json:"col_index"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type CreateTreeRequest struct {
	PlotID   string  `json:"plot_id"`
	TreeCode string  `json:"tree_code"`
	RowIndex int     `json:"row_index"`
	ColIndex int     `json:"col_index"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Status   string  `json:"status"`
}

type UpdateTreeRequest struct {
	TreeCode *string  `json:"tree_code"`
	RowIndex *int     `json:"row_index"`
	ColIndex *int     `json:"col_index"`
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Status   *string  `json:"status"`
}

type TreeListResponse struct {
	Items []TreeArchive `json:"items"`
}

type CreateWorkOrdersRequest struct {
	PlanID string `json:"plan_id"`
}

type WorkOrder struct {
	WorkOrderID    string                      `json:"work_order_id"`
	PlotID         string                      `json:"plot_id"`
	TreeID         string                      `json:"tree_id"`
	TreeCode       string                      `json:"tree_code"`
	PlanID         string                      `json:"plan_id"`
	Status         string                      `json:"status"`
	ZonePriorities []decision.ZonePriority     `json:"zone_priorities"`
	PickSequence   []decision.PickSequenceItem `json:"pick_sequence"`
	SkipItems      []decision.SkipItem         `json:"skip_items"`
	SkipReasonNote string                      `json:"skip_reason_note"`
	StartedAt      *string                     `json:"started_at"`
	CompletedAt    *string                     `json:"completed_at"`
	CreatedAt      string                      `json:"created_at"`
	UpdatedAt      string                      `json:"updated_at"`
}

type WorkOrderListResponse struct {
	Items []WorkOrder `json:"items"`
}

type UpdateWorkOrderRequest struct {
	Status         string  `json:"status"`
	SkipReasonNote *string `json:"skip_reason_note"`
}
