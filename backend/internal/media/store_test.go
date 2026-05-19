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

func TestMemoryStoreMarkMissingByLibrary(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	kept, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID: "library-1",
		Path:      "/media/keep.mkv",
	})
	if err != nil {
		t.Fatal(err)
	}
	missing, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID: "library-1",
		Path:      "/media/missing.mkv",
	})
	if err != nil {
		t.Fatal(err)
	}

	changed, err := store.MarkMissingByLibrary(context.Background(), "library-1", []string{kept.Path})
	if err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 missing file, got %d", changed)
	}

	found, ok, err := store.GetFileByPath(context.Background(), missing.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected missing file")
	}
	if found.Status != FileStatusMissing {
		t.Fatalf("expected missing status, got %s", found.Status)
	}

	missingFiles, err := store.ListFiles(context.Background(), FileQuery{
		LibraryID: "library-1",
		Status:    FileStatusMissing,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(missingFiles) != 1 {
		t.Fatalf("expected 1 missing file, got %d", len(missingFiles))
	}

	deleted, err := store.DeleteMissingByLibrary(context.Background(), "library-1")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted missing file, got %d", deleted)
	}
}
