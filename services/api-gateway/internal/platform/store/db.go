package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type DB struct {
	Conn   *sql.DB
	Driver string
}

func Open(cfg config.DBConfig) (*DB, error) {
	driverName, dsn, err := resolveDriver(cfg)
	if err != nil {
		return nil, err
	}

	if driverName == "sqlite" {
		if err := ensureSQLiteDir(dsn); err != nil {
			return nil, err
		}
	}

	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open gateway db: %w", err)
	}

	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeS) * time.Second)

	db := &DB{
		Conn:   conn,
		Driver: cfg.Driver,
	}

	if err := db.EnsureSchema(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}

func (db *DB) IsPostgres() bool {
	return db.Driver == "postgres"
}

func (db *DB) Placeholder(index int) string {
	if db.IsPostgres() {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func NewID(prefix string) (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(randomBytes), nil
}

func resolveDriver(cfg config.DBConfig) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "sqlite":
		return "sqlite", cfg.DSN, nil
	case "postgres":
		return "pgx", cfg.DSN, nil
	default:
		return "", "", fmt.Errorf("unsupported db driver %q", cfg.Driver)
	}
}

func ensureSQLiteDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
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
