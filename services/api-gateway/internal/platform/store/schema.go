package store

import (
	"database/sql"
	"fmt"
	"strings"
)

func (db *DB) EnsureSchema() error {
	statements := []string{
		`
CREATE TABLE IF NOT EXISTS decision_history (
	id TEXT PRIMARY KEY,
	entry_type TEXT NOT NULL DEFAULT 'tree_recommendation',
	created_at TEXT NOT NULL,
	request_json TEXT NOT NULL,
	response_json TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS plots (
	plot_id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	code TEXT NOT NULL UNIQUE,
	row_count INTEGER NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS trees (
	tree_id TEXT PRIMARY KEY,
	plot_id TEXT NOT NULL,
	tree_code TEXT NOT NULL,
	row_index INTEGER NOT NULL,
	col_index INTEGER NOT NULL,
	x REAL NOT NULL,
	y REAL NOT NULL,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS tree_observations (
	observation_id TEXT PRIMARY KEY,
	tree_id TEXT NOT NULL,
	captured_at TEXT NOT NULL,
	frame_index INTEGER NOT NULL,
	timestamp_ms INTEGER NOT NULL,
	frame_width INTEGER NOT NULL,
	frame_height INTEGER NOT NULL,
	detections_json TEXT NOT NULL,
	created_at TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS decision_plans (
	plan_id TEXT PRIMARY KEY,
	plot_id TEXT NOT NULL,
	generated_at TEXT NOT NULL,
	manual_override INTEGER NOT NULL DEFAULT 0,
	summary_json TEXT NOT NULL,
	tree_sequence_json TEXT NOT NULL,
	override_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS decision_plan_trees (
	entry_id TEXT PRIMARY KEY,
	plan_id TEXT NOT NULL,
	tree_id TEXT NOT NULL,
	priority_order INTEGER NOT NULL,
	recommendation_json TEXT NOT NULL
);
`,
		`
CREATE TABLE IF NOT EXISTS work_orders (
	work_order_id TEXT PRIMARY KEY,
	plot_id TEXT NOT NULL,
	tree_id TEXT NOT NULL,
	tree_code TEXT NOT NULL,
	plan_id TEXT NOT NULL,
	status TEXT NOT NULL,
	zone_priorities_json TEXT NOT NULL,
	pick_sequence_json TEXT NOT NULL,
	skip_items_json TEXT NOT NULL,
	skip_reason_note TEXT NOT NULL DEFAULT '',
	started_at TEXT,
	completed_at TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`,
		`CREATE INDEX IF NOT EXISTS idx_trees_plot_id ON trees(plot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tree_observations_tree_id ON tree_observations(tree_id);`,
		`CREATE INDEX IF NOT EXISTS idx_decision_plans_plot_id ON decision_plans(plot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_decision_plan_trees_plan_id ON decision_plan_trees(plan_id);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_plot_id ON work_orders(plot_id);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders(status);`,
	}

	for _, statement := range statements {
		if _, err := db.Conn.Exec(statement); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}

	if err := db.ensureDecisionHistoryEntryType(); err != nil {
		return err
	}

	return nil
}

func (db *DB) ensureDecisionHistoryEntryType() error {
	hasColumn, err := db.hasColumn("decision_history", "entry_type")
	if err != nil {
		return err
	}
	if hasColumn {
		return nil
	}

	statement := `ALTER TABLE decision_history ADD COLUMN entry_type TEXT NOT NULL DEFAULT 'tree_recommendation'`
	if db.IsPostgres() {
		statement = `ALTER TABLE decision_history ADD COLUMN IF NOT EXISTS entry_type TEXT NOT NULL DEFAULT 'tree_recommendation'`
	}
	if _, err := db.Conn.Exec(statement); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate column") || strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return nil
		}
		return fmt.Errorf("ensure decision_history.entry_type: %w", err)
	}
	return nil
}

func (db *DB) hasColumn(table string, column string) (bool, error) {
	if db.IsPostgres() {
		var exists bool
		err := db.Conn.QueryRow(
			`SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_name = $1 AND column_name = $2
			)`,
			table,
			column,
		).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("check postgres column %s.%s: %w", table, column, err)
		}
		return exists, nil
	}

	rows, err := db.Conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("check sqlite column %s.%s: %w", table, column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("scan sqlite column %s.%s: %w", table, column, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate sqlite columns %s.%s: %w", table, column, err)
	}
	return false, nil
}
