package decision

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Repository struct {
	db     *sql.DB
	driver string
}

func NewRepository(cfg config.DBConfig) (*Repository, error) {
	driverName, dsn, err := resolveDriver(cfg)
	if err != nil {
		return nil, err
	}

	if driverName == "sqlite" {
		if err := ensureSQLiteDir(dsn); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open decision db: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeS) * time.Second)

	repo := &Repository{
		db:     db,
		driver: cfg.Driver,
	}

	if err := repo.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Save(ctx context.Context, request SnapshotRequest, response RecommendationResponse) error {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal decision request: %w", err)
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal decision response: %w", err)
	}

	query := `INSERT INTO decision_history (id, created_at, request_json, response_json) VALUES (?, ?, ?, ?)`
	if r.driver == "postgres" {
		query = `INSERT INTO decision_history (id, created_at, request_json, response_json) VALUES ($1, $2, $3, $4)`
	}

	if _, err := r.db.ExecContext(
		ctx,
		query,
		response.DecisionID,
		response.CreatedAt,
		string(requestJSON),
		string(responseJSON),
	); err != nil {
		return fmt.Errorf("insert decision history: %w", err)
	}

	return nil
}

func (r *Repository) ListRecent(ctx context.Context, limit int) ([]HistoryItem, error) {
	query := fmt.Sprintf(
		`SELECT id, created_at, request_json, response_json FROM decision_history ORDER BY created_at DESC LIMIT %d`,
		limit,
	)

	rows, err := r.db.QueryContext(ctx, query)
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

func (r *Repository) ensureSchema() error {
	query := `
CREATE TABLE IF NOT EXISTS decision_history (
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL,
	request_json TEXT NOT NULL,
	response_json TEXT NOT NULL
);
`
	if _, err := r.db.Exec(query); err != nil {
		return fmt.Errorf("create decision_history table: %w", err)
	}
	return nil
}

func resolveDriver(cfg config.DBConfig) (string, string, error) {
	switch cfg.Driver {
	case "sqlite":
		return "sqlite", cfg.DSN, nil
	case "postgres":
		return "pgx", cfg.DSN, nil
	default:
		return "", "", fmt.Errorf("unsupported db driver %q", cfg.Driver)
	}
}

func ensureSQLiteDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" || len(dsn) >= 5 && dsn[:5] == "file:" {
		return nil
	}

	dir := filepath.Dir(dsn)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite directory %s: %w", dir, err)
	}
	return nil
}
