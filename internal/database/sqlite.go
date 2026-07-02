// Package database provides a SQLite database connection backed by
// modernc.org/sqlite (pure Go, no CGO required on Windows).
// Goose migrations are embedded and applied automatically at startup.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/compico/go-osu/internal/config"
	"github.com/compico/go-osu/migrations"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// DB wraps the standard *sql.DB with app-specific helpers.
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// New opens (or creates) the SQLite database at the path specified in cfg.
// The path is resolved relative to the running binary's directory so the
// database file always sits next to the executable.
func New(cfg *config.DatabaseConfig, logger *slog.Logger) (*DB, error) {
	path, err := resolvePath(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("database: resolve path: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("database: open %q: %w", path, err)
	}

	// SQLite performs best with a single writer connection.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	logger.Info("database opened", "path", path)
	return &DB{DB: db, logger: logger}, nil
}

// Migrate applies all pending goose migrations embedded in the migrations
// package. Safe to call on every startup — already-applied migrations are
// skipped automatically.
func (d *DB) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations.FS)

	// Silence goose's own output; we log the result ourselves.
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("database: set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, d.DB, "."); err != nil {
		return fmt.Errorf("database: migrate: %w", err)
	}

	d.logger.Info("database migrations applied")
	return nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.DB.Close()
}

// resolvePath returns an absolute path.
// If p is already absolute it is returned as-is; otherwise it is resolved
// relative to the directory containing the running executable.
func resolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(exe), p), nil
}
