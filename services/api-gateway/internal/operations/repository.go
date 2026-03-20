package operations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/decision"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
)

type Repository struct {
	db *store.DB
}

func NewRepository(db *store.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePlot(ctx context.Context, request CreatePlotRequest) (Plot, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	plotID, err := store.NewID("plot")
	if err != nil {
		return Plot{}, fmt.Errorf("new plot id: %w", err)
	}

	query := fmt.Sprintf(
		`INSERT INTO plots (plot_id, name, code, row_count, notes, created_at, updated_at)
		 VALUES (%s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
	)

	plot := Plot{
		PlotID:    plotID,
		Name:      strings.TrimSpace(request.Name),
		Code:      strings.TrimSpace(request.Code),
		RowCount:  request.RowCount,
		Notes:     request.Notes,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := r.db.Conn.ExecContext(ctx, query, plot.PlotID, plot.Name, plot.Code, plot.RowCount, plot.Notes, plot.CreatedAt, plot.UpdatedAt); err != nil {
		return Plot{}, fmt.Errorf("insert plot: %w", err)
	}
	return plot, nil
}

func (r *Repository) ListPlots(ctx context.Context) ([]Plot, error) {
	rows, err := r.db.Conn.QueryContext(ctx, `SELECT plot_id, name, code, row_count, notes, created_at, updated_at FROM plots ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query plots: %w", err)
	}
	defer rows.Close()

	items := []Plot{}
	for rows.Next() {
		var item Plot
		if err := rows.Scan(&item.PlotID, &item.Name, &item.Code, &item.RowCount, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan plot: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plots: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateTree(ctx context.Context, request CreateTreeRequest) (TreeArchive, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	treeID, err := store.NewID("tree")
	if err != nil {
		return TreeArchive{}, fmt.Errorf("new tree id: %w", err)
	}

	status := request.Status
	if status == "" {
		status = TreeStatusActive
	}

	query := fmt.Sprintf(
		`INSERT INTO trees (tree_id, plot_id, tree_code, row_index, col_index, x, y, status, created_at, updated_at)
		 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
		r.db.Placeholder(8),
		r.db.Placeholder(9),
		r.db.Placeholder(10),
	)

	tree := TreeArchive{
		TreeID:    treeID,
		PlotID:    request.PlotID,
		TreeCode:  strings.TrimSpace(request.TreeCode),
		RowIndex:  request.RowIndex,
		ColIndex:  request.ColIndex,
		X:         request.X,
		Y:         request.Y,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := r.db.Conn.ExecContext(ctx, query, tree.TreeID, tree.PlotID, tree.TreeCode, tree.RowIndex, tree.ColIndex, tree.X, tree.Y, tree.Status, tree.CreatedAt, tree.UpdatedAt); err != nil {
		return TreeArchive{}, fmt.Errorf("insert tree: %w", err)
	}
	return tree, nil
}

func (r *Repository) ListTreesByPlot(ctx context.Context, plotID string) ([]TreeArchive, error) {
	query := fmt.Sprintf(
		`SELECT tree_id, plot_id, tree_code, row_index, col_index, x, y, status, created_at, updated_at
		 FROM trees
		 WHERE plot_id = %s
		 ORDER BY row_index ASC, col_index ASC, tree_code ASC`,
		r.db.Placeholder(1),
	)
	rows, err := r.db.Conn.QueryContext(ctx, query, plotID)
	if err != nil {
		return nil, fmt.Errorf("query trees: %w", err)
	}
	defer rows.Close()

	items := []TreeArchive{}
	for rows.Next() {
		var item TreeArchive
		if err := rows.Scan(&item.TreeID, &item.PlotID, &item.TreeCode, &item.RowIndex, &item.ColIndex, &item.X, &item.Y, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tree: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trees: %w", err)
	}
	return items, nil
}

func (r *Repository) UpdateTree(ctx context.Context, treeID string, request UpdateTreeRequest) (TreeArchive, error) {
	current, err := r.loadTree(ctx, treeID)
	if err != nil {
		return TreeArchive{}, err
	}

	if request.TreeCode != nil {
		current.TreeCode = strings.TrimSpace(*request.TreeCode)
	}
	if request.RowIndex != nil {
		current.RowIndex = *request.RowIndex
	}
	if request.ColIndex != nil {
		current.ColIndex = *request.ColIndex
	}
	if request.X != nil {
		current.X = *request.X
	}
	if request.Y != nil {
		current.Y = *request.Y
	}
	if request.Status != nil {
		current.Status = *request.Status
	}
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	query := fmt.Sprintf(
		`UPDATE trees
		 SET tree_code = %s, row_index = %s, col_index = %s, x = %s, y = %s, status = %s, updated_at = %s
		 WHERE tree_id = %s`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
		r.db.Placeholder(8),
	)

	if _, err := r.db.Conn.ExecContext(ctx, query, current.TreeCode, current.RowIndex, current.ColIndex, current.X, current.Y, current.Status, current.UpdatedAt, current.TreeID); err != nil {
		return TreeArchive{}, fmt.Errorf("update tree: %w", err)
	}
	return current, nil
}

func (r *Repository) CreateWorkOrdersFromPlan(ctx context.Context, planID string) ([]WorkOrder, error) {
	tx, err := r.db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create work orders tx: %w", err)
	}
	defer tx.Rollback()

	plan, err := r.loadPlanForWorkOrders(ctx, tx, planID)
	if err != nil {
		return nil, err
	}

	items := []WorkOrder{}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, recommendation := range plan.TreeRecommendations {
		if recommendation.Status != decision.TreeRecommendationReady {
			continue
		}

		var existingID string
		checkQuery := fmt.Sprintf(
			`SELECT work_order_id FROM work_orders WHERE plan_id = %s AND tree_id = %s LIMIT 1`,
			r.db.Placeholder(1),
			r.db.Placeholder(2),
		)
		err := tx.QueryRowContext(ctx, checkQuery, plan.PlanID, recommendation.TreeID).Scan(&existingID)
		if err == nil {
			existing, loadErr := r.loadWorkOrderTx(ctx, tx, existingID)
			if loadErr != nil {
				return nil, loadErr
			}
			items = append(items, existing)
			continue
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("check existing work order: %w", err)
		}

		workOrderID, err := store.NewID("work_order")
		if err != nil {
			return nil, fmt.Errorf("new work order id: %w", err)
		}

		zoneJSON, _ := json.Marshal(recommendation.ZonePriorities)
		pickJSON, _ := json.Marshal(recommendation.PickSequence)
		skipJSON, _ := json.Marshal(recommendation.SkipItems)

		insertQuery := fmt.Sprintf(
			`INSERT INTO work_orders
			 (work_order_id, plot_id, tree_id, tree_code, plan_id, status, zone_priorities_json, pick_sequence_json, skip_items_json, skip_reason_note, started_at, completed_at, created_at, updated_at)
			 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
			r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4), r.db.Placeholder(5), r.db.Placeholder(6),
			r.db.Placeholder(7), r.db.Placeholder(8), r.db.Placeholder(9), r.db.Placeholder(10), r.db.Placeholder(11), r.db.Placeholder(12), r.db.Placeholder(13), r.db.Placeholder(14),
		)
		if _, err := tx.ExecContext(ctx, insertQuery, workOrderID, plan.PlotID, recommendation.TreeID, recommendation.TreeCode, plan.PlanID, WorkOrderStatusPending, string(zoneJSON), string(pickJSON), string(skipJSON), "", nil, nil, now, now); err != nil {
			return nil, fmt.Errorf("insert work order: %w", err)
		}

		items = append(items, WorkOrder{
			WorkOrderID:    workOrderID,
			PlotID:         plan.PlotID,
			TreeID:         recommendation.TreeID,
			TreeCode:       recommendation.TreeCode,
			PlanID:         plan.PlanID,
			Status:         WorkOrderStatusPending,
			ZonePriorities: recommendation.ZonePriorities,
			PickSequence:   recommendation.PickSequence,
			SkipItems:      recommendation.SkipItems,
			SkipReasonNote: "",
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create work orders: %w", err)
	}
	return items, nil
}

func (r *Repository) ListWorkOrders(ctx context.Context, plotID string, status string) ([]WorkOrder, error) {
	args := []any{}
	conditions := []string{}
	if plotID != "" {
		conditions = append(conditions, fmt.Sprintf("plot_id = %s", r.db.Placeholder(len(args)+1)))
		args = append(args, plotID)
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = %s", r.db.Placeholder(len(args)+1)))
		args = append(args, status)
	}

	query := `SELECT work_order_id, plot_id, tree_id, tree_code, plan_id, status, zone_priorities_json, pick_sequence_json, skip_items_json, skip_reason_note, started_at, completed_at, created_at, updated_at FROM work_orders`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query work orders: %w", err)
	}
	defer rows.Close()

	items := []WorkOrder{}
	for rows.Next() {
		item, err := scanWorkOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate work orders: %w", err)
	}
	return items, nil
}

func (r *Repository) UpdateWorkOrder(ctx context.Context, workOrderID string, request UpdateWorkOrderRequest) (WorkOrder, error) {
	current, err := r.loadWorkOrder(ctx, workOrderID)
	if err != nil {
		return WorkOrder{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	current.Status = request.Status
	current.UpdatedAt = now
	if request.SkipReasonNote != nil {
		current.SkipReasonNote = *request.SkipReasonNote
	}
	switch request.Status {
	case WorkOrderStatusInProgress:
		current.StartedAt = &now
		current.CompletedAt = nil
	case WorkOrderStatusCompleted:
		if current.StartedAt == nil {
			current.StartedAt = &now
		}
		current.CompletedAt = &now
	case WorkOrderStatusSkipped:
		current.CompletedAt = &now
	default:
		current.StartedAt = nil
		current.CompletedAt = nil
	}

	query := fmt.Sprintf(
		`UPDATE work_orders
		 SET status = %s, skip_reason_note = %s, started_at = %s, completed_at = %s, updated_at = %s
		 WHERE work_order_id = %s`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4), r.db.Placeholder(5), r.db.Placeholder(6),
	)

	if _, err := r.db.Conn.ExecContext(ctx, query, current.Status, current.SkipReasonNote, current.StartedAt, current.CompletedAt, current.UpdatedAt, current.WorkOrderID); err != nil {
		return WorkOrder{}, fmt.Errorf("update work order: %w", err)
	}
	return current, nil
}

func (r *Repository) loadTree(ctx context.Context, treeID string) (TreeArchive, error) {
	query := fmt.Sprintf(
		`SELECT tree_id, plot_id, tree_code, row_index, col_index, x, y, status, created_at, updated_at
		 FROM trees
		 WHERE tree_id = %s`,
		r.db.Placeholder(1),
	)
	row := r.db.Conn.QueryRowContext(ctx, query, treeID)
	var item TreeArchive
	if err := row.Scan(&item.TreeID, &item.PlotID, &item.TreeCode, &item.RowIndex, &item.ColIndex, &item.X, &item.Y, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return TreeArchive{}, fmt.Errorf("tree %s not found", treeID)
		}
		return TreeArchive{}, fmt.Errorf("scan tree: %w", err)
	}
	return item, nil
}

func (r *Repository) loadPlanForWorkOrders(ctx context.Context, tx *sql.Tx, planID string) (decision.DecisionPlan, error) {
	query := fmt.Sprintf(
		`SELECT plot_id, generated_at, manual_override, summary_json, tree_sequence_json, override_json
		 FROM decision_plans WHERE plan_id = %s`,
		r.db.Placeholder(1),
	)
	row := tx.QueryRowContext(ctx, query, planID)

	var (
		plan             decision.DecisionPlan
		manualOverride   any
		summaryJSON      string
		treeSequenceJSON string
		overrideJSON     string
	)
	plan.PlanID = planID
	if err := row.Scan(&plan.PlotID, &plan.GeneratedAt, &manualOverride, &summaryJSON, &treeSequenceJSON, &overrideJSON); err != nil {
		if err == sql.ErrNoRows {
			return decision.DecisionPlan{}, fmt.Errorf("plan %s not found", planID)
		}
		return decision.DecisionPlan{}, fmt.Errorf("scan plan for work orders: %w", err)
	}
	if err := json.Unmarshal([]byte(summaryJSON), &plan.Summary); err != nil {
		return decision.DecisionPlan{}, fmt.Errorf("decode plan summary: %w", err)
	}
	if err := json.Unmarshal([]byte(treeSequenceJSON), &plan.TreeSequence); err != nil {
		return decision.DecisionPlan{}, fmt.Errorf("decode plan tree sequence: %w", err)
	}
	if overrideJSON != "" {
		if err := json.Unmarshal([]byte(overrideJSON), &plan.ManualOverrideState); err != nil {
			return decision.DecisionPlan{}, fmt.Errorf("decode plan override: %w", err)
		}
	}
	plan.ManualOverride = storageToBool(manualOverride)

	recommendationQuery := fmt.Sprintf(`SELECT recommendation_json FROM decision_plan_trees WHERE plan_id = %s ORDER BY priority_order ASC`, r.db.Placeholder(1))
	rows, err := tx.QueryContext(ctx, recommendationQuery, planID)
	if err != nil {
		return decision.DecisionPlan{}, fmt.Errorf("query plan tree recommendations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var recommendationJSON string
		if err := rows.Scan(&recommendationJSON); err != nil {
			return decision.DecisionPlan{}, fmt.Errorf("scan plan tree recommendation: %w", err)
		}
		var recommendation decision.TreeRecommendation
		if err := json.Unmarshal([]byte(recommendationJSON), &recommendation); err != nil {
			return decision.DecisionPlan{}, fmt.Errorf("decode plan tree recommendation: %w", err)
		}
		plan.TreeRecommendations = append(plan.TreeRecommendations, recommendation)
	}
	if err := rows.Err(); err != nil {
		return decision.DecisionPlan{}, fmt.Errorf("iterate plan tree recommendations: %w", err)
	}
	return plan, nil
}

func (r *Repository) loadWorkOrder(ctx context.Context, workOrderID string) (WorkOrder, error) {
	query := fmt.Sprintf(
		`SELECT work_order_id, plot_id, tree_id, tree_code, plan_id, status, zone_priorities_json, pick_sequence_json, skip_items_json, skip_reason_note, started_at, completed_at, created_at, updated_at
		 FROM work_orders WHERE work_order_id = %s`,
		r.db.Placeholder(1),
	)
	row := r.db.Conn.QueryRowContext(ctx, query, workOrderID)
	return scanWorkOrder(row)
}

func (r *Repository) loadWorkOrderTx(ctx context.Context, tx *sql.Tx, workOrderID string) (WorkOrder, error) {
	query := fmt.Sprintf(
		`SELECT work_order_id, plot_id, tree_id, tree_code, plan_id, status, zone_priorities_json, pick_sequence_json, skip_items_json, skip_reason_note, started_at, completed_at, created_at, updated_at
		 FROM work_orders WHERE work_order_id = %s`,
		r.db.Placeholder(1),
	)
	row := tx.QueryRowContext(ctx, query, workOrderID)
	return scanWorkOrder(row)
}

func scanWorkOrder(scanner interface{ Scan(dest ...any) error }) (WorkOrder, error) {
	var (
		item        WorkOrder
		zoneJSON    string
		pickJSON    string
		skipJSON    string
		startedAt   sql.NullString
		completedAt sql.NullString
	)
	if err := scanner.Scan(
		&item.WorkOrderID,
		&item.PlotID,
		&item.TreeID,
		&item.TreeCode,
		&item.PlanID,
		&item.Status,
		&zoneJSON,
		&pickJSON,
		&skipJSON,
		&item.SkipReasonNote,
		&startedAt,
		&completedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return WorkOrder{}, fmt.Errorf("work order not found")
		}
		return WorkOrder{}, fmt.Errorf("scan work order: %w", err)
	}
	if zoneJSON != "" {
		if err := json.Unmarshal([]byte(zoneJSON), &item.ZonePriorities); err != nil {
			return WorkOrder{}, fmt.Errorf("decode work order zones: %w", err)
		}
	}
	if pickJSON != "" {
		if err := json.Unmarshal([]byte(pickJSON), &item.PickSequence); err != nil {
			return WorkOrder{}, fmt.Errorf("decode work order pick sequence: %w", err)
		}
	}
	if skipJSON != "" {
		if err := json.Unmarshal([]byte(skipJSON), &item.SkipItems); err != nil {
			return WorkOrder{}, fmt.Errorf("decode work order skip items: %w", err)
		}
	}
	if startedAt.Valid {
		item.StartedAt = &startedAt.String
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.String
	}
	return item, nil
}

func storageToBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int64:
		return typed != 0
	case int:
		return typed != 0
	case []byte:
		return string(typed) == "1" || string(typed) == "true"
	case string:
		return typed == "1" || typed == "true"
	default:
		return false
	}
}
