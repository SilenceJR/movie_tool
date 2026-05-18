package database

import (
	"context"
	"strings"
	"testing"
)

func TestOpenSQLiteRequiresPath(t *testing.T) {
	_, err := OpenSQLite(context.Background(), SQLiteOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected path error, got %v", err)
	}
}
