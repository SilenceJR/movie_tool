package media

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreUpsertFile(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	file, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID:         "library-1",
		Path:              "/media/Inception.2010.mkv",
		Size:              12,
		ModifiedAt:        now,
		DetectedMediaType: "movie",
		ParsedTitle:       "Inception",
		ParsedYear:        2010,
	})
	if err != nil {
		t.Fatal(err)
	}
	if file.Status != FileStatusAvailable {
		t.Fatalf("expected available status, got %s", file.Status)
	}

	updated, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID:         "library-1",
		Path:              "/media/Inception.2010.mkv",
		Size:              24,
		ModifiedAt:        now.Add(time.Hour),
		DetectedMediaType: "movie",
		ParsedTitle:       "Inception",
		ParsedYear:        2010,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != file.ID {
		t.Fatalf("expected upsert to keep id %q, got %q", file.ID, updated.ID)
	}
	if updated.Size != 24 {
		t.Fatalf("expected updated size 24, got %d", updated.Size)
	}
}
