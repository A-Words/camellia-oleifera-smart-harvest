package decision

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
)

const (
	historyEntryTypeTreeRecommendation = "tree_recommendation"
	historyEntryTypePlanAudit          = "plan_audit"
)

type Repository struct {
	db *store.DB
}

func NewRepository(db *store.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveTreeHistory(ctx context.Context, request SnapshotRequest, response RecommendationResponse) error {
	return r.saveHistory(ctx, historyEntryTypeTreeRecommendation, response.DecisionID, response.CreatedAt, request, response)
}

func (r *Repository) SavePlanAudit(ctx context.Context, plan DecisionPlan) error {
	request := map[string]string{
		"plot_id": plan.PlotID,
	}
	return r.saveHistory(ctx, historyEntryTypePlanAudit, plan.PlanID, plan.GeneratedAt, request, plan)
}

func (r *Repository) ListRecentTreeHistory(ctx context.Context, limit int) ([]HistoryItem, error) {
	query := fmt.Sprintf(
		`SELECT id, created_at, request_json, response_json
		 FROM decision_history
		 WHERE entry_type = %s
		 ORDER BY created_at DESC
		 LIMIT %d`,
		r.db.Placeholder(1),
		limit,
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, historyEntryTypeTreeRecommendation)
	if err != nil {
		return nil, fmt.Errorf("query decision history: %w", err)
	}
	defer rows.Close()

	items := make([]HistoryItem, 0, limit)
	for rows.Next() {
		var (
			id           string
			createdAt    string
			requestJSON  string
			responseJSON string
		)
		if err := rows.Scan(&id, &createdAt, &requestJSON, &responseJSON); err != nil {
			return nil, fmt.Errorf("scan decision history: %w", err)
		}

		var request SnapshotRequest
		if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
			return nil, fmt.Errorf("decode decision request: %w", err)
		}

		var response RecommendationResponse
		if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
			return nil, fmt.Errorf("decode decision response: %w", err)
		}

		items = append(items, HistoryItem{
			DecisionID:     id,
			CreatedAt:      createdAt,
			Request:        request,
			Summary:        response.Summary,
			ZonePriorities: response.ZonePriorities,
			PickSequence:   response.PickSequence,
			SkipItems:      response.SkipItems,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decision history: %w", err)
	}

	return items, nil
}

func (r *Repository) CreateObservation(ctx context.Context, observation Observation) error {
	detectionsJSON, err := json.Marshal(observation.Detections)
	if err != nil {
		return fmt.Errorf("marshal observation detections: %w", err)
	}

	query := fmt.Sprintf(
		`INSERT INTO tree_observations
		 (observation_id, tree_id, captured_at, frame_index, timestamp_ms, frame_width, frame_height, detections_json, created_at)
		 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
		r.db.Placeholder(8),
		r.db.Placeholder(9),
	)

	if _, err := r.db.Conn.ExecContext(
		ctx,
		query,
		observation.ObservationID,
		observation.TreeID,
		observation.CapturedAt,
		observation.FrameIndex,
		observation.TimestampMS,
		observation.FrameWidth,
		observation.FrameHeight,
		string(detectionsJSON),
		observation.CapturedAt,
	); err != nil {
		return fmt.Errorf("insert observation: %w", err)
	}

	return nil
}

func (r *Repository) ListObservationsByTree(ctx context.Context, treeID string) ([]Observation, error) {
	query := fmt.Sprintf(
		`SELECT observation_id, tree_id, captured_at, frame_index, timestamp_ms, frame_width, frame_height, detections_json
		 FROM tree_observations
		 WHERE tree_id = %s
		 ORDER BY captured_at DESC`,
		r.db.Placeholder(1),
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, treeID)
	if err != nil {
		return nil, fmt.Errorf("query observations: %w", err)
	}
	defer rows.Close()

	items := []Observation{}
	for rows.Next() {
		item, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observations: %w", err)
	}
	return items, nil
}

func (r *Repository) LoadLatestObservationsForPlot(ctx context.Context, plotID string) ([]latestObservation, error) {
	query := fmt.Sprintf(
		`SELECT
			t.tree_id,
			t.plot_id,
			t.tree_code,
			t.row_index,
			t.col_index,
			t.x,
			t.y,
			t.status,
			o.observation_id,
			o.tree_id,
			o.captured_at,
			o.frame_index,
			o.timestamp_ms,
			o.frame_width,
			o.frame_height,
			o.detections_json
		 FROM trees t
		 LEFT JOIN tree_observations o
		   ON o.observation_id = (
			 SELECT o2.observation_id
			 FROM tree_observations o2
			 WHERE o2.tree_id = t.tree_id
			 ORDER BY o2.captured_at DESC
			 LIMIT 1
		   )
		 WHERE t.plot_id = %s
		 ORDER BY t.row_index ASC, t.col_index ASC, t.tree_code ASC`,
		r.db.Placeholder(1),
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, plotID)
	if err != nil {
		return nil, fmt.Errorf("query latest observations: %w", err)
	}
	defer rows.Close()

	items := []latestObservation{}
	for rows.Next() {
		var (
			tree           TreeMeta
			observationID  sql.NullString
			obsTreeID      sql.NullString
			capturedAt     sql.NullString
			frameIndex     sql.NullInt64
			timestampMS    sql.NullInt64
			frameWidth     sql.NullInt64
			frameHeight    sql.NullInt64
			detectionsJSON sql.NullString
		)
		if err := rows.Scan(
			&tree.TreeID,
			&tree.PlotID,
			&tree.TreeCode,
			&tree.RowIndex,
			&tree.ColIndex,
			&tree.X,
			&tree.Y,
			&tree.Status,
			&observationID,
			&obsTreeID,
			&capturedAt,
			&frameIndex,
			&timestampMS,
			&frameWidth,
			&frameHeight,
			&detectionsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan latest observations: %w", err)
		}

		item := latestObservation{Tree: tree}
		if observationID.Valid {
			detections := []Detection{}
			if detectionsJSON.Valid && detectionsJSON.String != "" {
				if err := json.Unmarshal([]byte(detectionsJSON.String), &detections); err != nil {
					return nil, fmt.Errorf("decode latest observation detections: %w", err)
				}
			}
			item.Observation = &Observation{
				ObservationID: observationID.String,
				TreeID:        obsTreeID.String,
				CapturedAt:    capturedAt.String,
				FrameIndex:    int(frameIndex.Int64),
				TimestampMS:   timestampMS.Int64,
				FrameWidth:    int(frameWidth.Int64),
				FrameHeight:   int(frameHeight.Int64),
				Detections:    detections,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest observations: %w", err)
	}
	return items, nil
}

func (r *Repository) SavePlan(ctx context.Context, plan DecisionPlan) error {
	summaryJSON, err := json.Marshal(plan.Summary)
	if err != nil {
		return fmt.Errorf("marshal plan summary: %w", err)
	}
	treeSequenceJSON, err := json.Marshal(plan.TreeSequence)
	if err != nil {
		return fmt.Errorf("marshal tree sequence: %w", err)
	}
	overrideJSON, err := json.Marshal(plan.ManualOverrideState)
	if err != nil {
		return fmt.Errorf("marshal manual override: %w", err)
	}

	tx, err := r.db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save plan tx: %w", err)
	}
	defer tx.Rollback()

	insertPlanQuery := fmt.Sprintf(
		`INSERT INTO decision_plans
		 (plan_id, plot_id, generated_at, manual_override, summary_json, tree_sequence_json, override_json, created_at, updated_at)
		 VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
		r.db.Placeholder(8),
		r.db.Placeholder(9),
	)

	if _, err := tx.ExecContext(
		ctx,
		insertPlanQuery,
		plan.PlanID,
		plan.PlotID,
		plan.GeneratedAt,
		boolToStorage(plan.ManualOverride),
		string(summaryJSON),
		string(treeSequenceJSON),
		string(overrideJSON),
		plan.GeneratedAt,
		plan.GeneratedAt,
	); err != nil {
		return fmt.Errorf("insert decision plan: %w", err)
	}

	if err := r.replacePlanTrees(ctx, tx, plan); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save plan: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePlan(ctx context.Context, plan DecisionPlan) error {
	summaryJSON, err := json.Marshal(plan.Summary)
	if err != nil {
		return fmt.Errorf("marshal updated plan summary: %w", err)
	}
	treeSequenceJSON, err := json.Marshal(plan.TreeSequence)
	if err != nil {
		return fmt.Errorf("marshal updated tree sequence: %w", err)
	}
	overrideJSON, err := json.Marshal(plan.ManualOverrideState)
	if err != nil {
		return fmt.Errorf("marshal updated override: %w", err)
	}

	tx, err := r.db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update plan tx: %w", err)
	}
	defer tx.Rollback()

	updateQuery := fmt.Sprintf(
		`UPDATE decision_plans
		 SET generated_at = %s, manual_override = %s, summary_json = %s, tree_sequence_json = %s, override_json = %s, updated_at = %s
		 WHERE plan_id = %s`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
		r.db.Placeholder(6),
		r.db.Placeholder(7),
	)

	if _, err := tx.ExecContext(
		ctx,
		updateQuery,
		plan.GeneratedAt,
		boolToStorage(plan.ManualOverride),
		string(summaryJSON),
		string(treeSequenceJSON),
		string(overrideJSON),
		plan.GeneratedAt,
		plan.PlanID,
	); err != nil {
		return fmt.Errorf("update decision plan: %w", err)
	}

	deleteQuery := fmt.Sprintf(`DELETE FROM decision_plan_trees WHERE plan_id = %s`, r.db.Placeholder(1))
	if _, err := tx.ExecContext(ctx, deleteQuery, plan.PlanID); err != nil {
		return fmt.Errorf("delete decision plan trees: %w", err)
	}

	if err := r.replacePlanTrees(ctx, tx, plan); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update plan: %w", err)
	}
	return nil
}

func (r *Repository) ListPlansByPlot(ctx context.Context, plotID string) ([]DecisionPlan, error) {
	query := fmt.Sprintf(
		`SELECT plan_id, plot_id, generated_at, manual_override, summary_json, tree_sequence_json, override_json
		 FROM decision_plans
		 WHERE plot_id = %s
		 ORDER BY generated_at DESC`,
		r.db.Placeholder(1),
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, plotID)
	if err != nil {
		return nil, fmt.Errorf("query plans: %w", err)
	}
	defer rows.Close()

	items := []DecisionPlan{}
	for rows.Next() {
		plan, err := scanPlanRow(rows)
		if err != nil {
			return nil, err
		}

		recommendations, err := r.loadPlanRecommendations(ctx, plan.PlanID)
		if err != nil {
			return nil, err
		}
		plan.TreeRecommendations = recommendations
		items = append(items, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}
	return items, nil
}

func (r *Repository) LoadPlan(ctx context.Context, planID string) (DecisionPlan, error) {
	query := fmt.Sprintf(
		`SELECT plan_id, plot_id, generated_at, manual_override, summary_json, tree_sequence_json, override_json
		 FROM decision_plans
		 WHERE plan_id = %s`,
		r.db.Placeholder(1),
	)

	row := r.db.Conn.QueryRowContext(ctx, query, planID)
	plan, err := scanPlanRow(row)
	if err != nil {
		return DecisionPlan{}, err
	}

	recommendations, err := r.loadPlanRecommendations(ctx, plan.PlanID)
	if err != nil {
		return DecisionPlan{}, err
	}
	plan.TreeRecommendations = recommendations
	return plan, nil
}

func (r *Repository) LoadTreeMetaForPlot(ctx context.Context, plotID string) (map[string]TreeMeta, error) {
	query := fmt.Sprintf(
		`SELECT tree_id, plot_id, tree_code, row_index, col_index, x, y, status
		 FROM trees
		 WHERE plot_id = %s`,
		r.db.Placeholder(1),
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, plotID)
	if err != nil {
		return nil, fmt.Errorf("query tree meta: %w", err)
	}
	defer rows.Close()

	items := map[string]TreeMeta{}
	for rows.Next() {
		var meta TreeMeta
		if err := rows.Scan(&meta.TreeID, &meta.PlotID, &meta.TreeCode, &meta.RowIndex, &meta.ColIndex, &meta.X, &meta.Y, &meta.Status); err != nil {
			return nil, fmt.Errorf("scan tree meta: %w", err)
		}
		items[meta.TreeID] = meta
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tree meta: %w", err)
	}
	return items, nil
}

func (r *Repository) replacePlanTrees(ctx context.Context, tx *sql.Tx, plan DecisionPlan) error {
	insertTreeQuery := fmt.Sprintf(
		`INSERT INTO decision_plan_trees (entry_id, plan_id, tree_id, priority_order, recommendation_json)
		 VALUES (%s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
	)

	priorityByTree := map[string]int{}
	for _, item := range plan.TreeSequence {
		priorityByTree[item.TreeID] = item.PriorityOrder
	}

	for _, recommendation := range plan.TreeRecommendations {
		entryID, err := store.NewID("plan_tree")
		if err != nil {
			return fmt.Errorf("new plan tree id: %w", err)
		}

		recommendationJSON, err := json.Marshal(recommendation)
		if err != nil {
			return fmt.Errorf("marshal tree recommendation: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			insertTreeQuery,
			entryID,
			plan.PlanID,
			recommendation.TreeID,
			priorityByTree[recommendation.TreeID],
			string(recommendationJSON),
		); err != nil {
			return fmt.Errorf("insert decision plan tree: %w", err)
		}
	}

	return nil
}

func (r *Repository) loadPlanRecommendations(ctx context.Context, planID string) ([]TreeRecommendation, error) {
	query := fmt.Sprintf(
		`SELECT recommendation_json
		 FROM decision_plan_trees
		 WHERE plan_id = %s
		 ORDER BY priority_order ASC, tree_id ASC`,
		r.db.Placeholder(1),
	)

	rows, err := r.db.Conn.QueryContext(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("query plan tree recommendations: %w", err)
	}
	defer rows.Close()

	items := []TreeRecommendation{}
	for rows.Next() {
		var recommendationJSON string
		if err := rows.Scan(&recommendationJSON); err != nil {
			return nil, fmt.Errorf("scan plan tree recommendation: %w", err)
		}
		var recommendation TreeRecommendation
		if err := json.Unmarshal([]byte(recommendationJSON), &recommendation); err != nil {
			return nil, fmt.Errorf("decode plan tree recommendation: %w", err)
		}
		items = append(items, recommendation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan tree recommendations: %w", err)
	}
	return items, nil
}

func (r *Repository) saveHistory(ctx context.Context, entryType string, id string, createdAt string, request any, response any) error {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal history request: %w", err)
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal history response: %w", err)
	}

	query := fmt.Sprintf(
		`INSERT INTO decision_history (id, entry_type, created_at, request_json, response_json)
		 VALUES (%s, %s, %s, %s, %s)`,
		r.db.Placeholder(1),
		r.db.Placeholder(2),
		r.db.Placeholder(3),
		r.db.Placeholder(4),
		r.db.Placeholder(5),
	)

	if _, err := r.db.Conn.ExecContext(ctx, query, id, entryType, createdAt, string(requestJSON), string(responseJSON)); err != nil {
		return fmt.Errorf("insert decision history: %w", err)
	}

	return nil
}

func scanObservation(scanner interface{ Scan(dest ...any) error }) (Observation, error) {
	var (
		observation    Observation
		detectionsJSON string
	)
	if err := scanner.Scan(
		&observation.ObservationID,
		&observation.TreeID,
		&observation.CapturedAt,
		&observation.FrameIndex,
		&observation.TimestampMS,
		&observation.FrameWidth,
		&observation.FrameHeight,
		&detectionsJSON,
	); err != nil {
		return Observation{}, fmt.Errorf("scan observation: %w", err)
	}
	if detectionsJSON != "" {
		if err := json.Unmarshal([]byte(detectionsJSON), &observation.Detections); err != nil {
			return Observation{}, fmt.Errorf("decode observation detections: %w", err)
		}
	}
	return observation, nil
}

func scanPlanRow(scanner interface{ Scan(dest ...any) error }) (DecisionPlan, error) {
	var (
		plan             DecisionPlan
		manualOverride   any
		summaryJSON      string
		treeSequenceJSON string
		overrideJSON     string
	)
	if err := scanner.Scan(&plan.PlanID, &plan.PlotID, &plan.GeneratedAt, &manualOverride, &summaryJSON, &treeSequenceJSON, &overrideJSON); err != nil {
		return DecisionPlan{}, fmt.Errorf("scan plan row: %w", err)
	}
	if err := json.Unmarshal([]byte(summaryJSON), &plan.Summary); err != nil {
		return DecisionPlan{}, fmt.Errorf("decode plan summary: %w", err)
	}
	if err := json.Unmarshal([]byte(treeSequenceJSON), &plan.TreeSequence); err != nil {
		return DecisionPlan{}, fmt.Errorf("decode plan tree sequence: %w", err)
	}
	if overrideJSON != "" {
		if err := json.Unmarshal([]byte(overrideJSON), &plan.ManualOverrideState); err != nil {
			return DecisionPlan{}, fmt.Errorf("decode manual override: %w", err)
		}
	}
	plan.ManualOverride = storageToBool(manualOverride)
	return plan, nil
}

func boolToStorage(value bool) any {
	if value {
		return 1
	}
	return 0
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
