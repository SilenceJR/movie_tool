package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
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

	db, err := sql.Open("sqlite3", options.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite wal: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set sqlite busy timeout: %w", err)
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
