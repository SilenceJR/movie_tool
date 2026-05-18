package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestMigrationsEmbedded(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one migration")
	}
	if migrations[0].Name != "0001_initial.sql" {
		t.Fatalf("expected first migration 0001_initial.sql, got %q", migrations[0].Name)
	}
	if migrations[0].SQL == "" {
		t.Fatal("expected migration SQL")
	}
}

func TestRunnerAppliesMigrationsInNameOrder(t *testing.T) {
	db := newFakeDB(nil)
	runner := NewRunner(db, []Migration{
		{Name: "0002_second.sql", SQL: "second"},
		{Name: "0001_first.sql", SQL: "first"},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"create schema_migrations",
		"query schema_migrations",
		"exec first",
		"record 0001_first.sql",
		"exec second",
		"record 0002_second.sql",
	}
	if got := db.calls; !equalStrings(got, want) {
		t.Fatalf("calls mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestRunnerSkipsAppliedMigrations(t *testing.T) {
	db := newFakeDB(map[string]bool{"0001_first.sql": true})
	runner := NewRunner(db, []Migration{
		{Name: "0001_first.sql", SQL: "first"},
		{Name: "0002_second.sql", SQL: "second"},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"create schema_migrations",
		"query schema_migrations",
		"exec second",
		"record 0002_second.sql",
	}
	if got := db.calls; !equalStrings(got, want) {
		t.Fatalf("calls mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestRunnerStopsOnMigrationFailure(t *testing.T) {
	db := newFakeDB(nil)
	db.failSQL = "second"
	runner := NewRunner(db, []Migration{
		{Name: "0001_first.sql", SQL: "first"},
		{Name: "0002_second.sql", SQL: "second"},
		{Name: "0003_third.sql", SQL: "third"},
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "0002_second.sql") {
		t.Fatalf("expected migration name in error, got %v", err)
	}

	want := []string{
		"create schema_migrations",
		"query schema_migrations",
		"exec first",
		"record 0001_first.sql",
		"exec second",
	}
	if got := db.calls; !equalStrings(got, want) {
		t.Fatalf("calls mismatch\nwant: %#v\n got: %#v", want, got)
	}
	if db.applied["0002_second.sql"] {
		t.Fatal("failed migration should not be recorded")
	}
}

type fakeDB struct {
	applied map[string]bool
	calls   []string
	failSQL string
}

func newFakeDB(applied map[string]bool) *fakeDB {
	copied := make(map[string]bool)
	for version, ok := range applied {
		copied[version] = ok
	}
	return &fakeDB{applied: copied}
}

func (db *fakeDB) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	switch {
	case strings.Contains(query, "CREATE TABLE IF NOT EXISTS schema_migrations"):
		db.calls = append(db.calls, "create schema_migrations")
	case strings.Contains(query, "INSERT INTO schema_migrations"):
		version := args[0].(string)
		db.calls = append(db.calls, "record "+version)
		db.applied[version] = true
	default:
		db.calls = append(db.calls, "exec "+query)
		if query == db.failSQL {
			return nil, errors.New("migration failed")
		}
	}
	return fakeResult(0), nil
}

func (db *fakeDB) QueryContext(_ context.Context, query string, _ ...any) (Rows, error) {
	if query != `SELECT version FROM schema_migrations` {
		return nil, errors.New("unexpected query")
	}
	db.calls = append(db.calls, "query schema_migrations")

	versions := make([]string, 0, len(db.applied))
	for version, ok := range db.applied {
		if ok {
			versions = append(versions, version)
		}
	}
	return &fakeRows{values: versions}, nil
}

type fakeRows struct {
	values []string
	index  int
}

func (r *fakeRows) Next() bool {
	return r.index < len(r.values)
}

func (r *fakeRows) Scan(dest ...any) error {
	value, ok := dest[0].(*string)
	if !ok {
		return errors.New("expected string destination")
	}
	*value = r.values[r.index]
	r.index++
	return nil
}

func (r *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Err() error {
	return nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) {
	return int64(r), nil
}

func (r fakeResult) RowsAffected() (int64, error) {
	return int64(r), nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
