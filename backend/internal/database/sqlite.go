package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

type SQLiteOptions struct {
	Path string
}

func OpenSQLite(ctx context.Context, options SQLiteOptions) (*sql.DB, error) {
	if options.Path == "" {
		return nil, fmt.Errorf("sqlite database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	runner, err := NewEmbeddedRunner(sqlAdapter{db: db})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := runner.Run(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

type sqlAdapter struct {
	db *sql.DB
}

func (a sqlAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

func (a sqlAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}
