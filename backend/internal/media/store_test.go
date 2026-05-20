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

func TestMemoryStoreMarkFileFailedAndClearOnUpsert(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	failed, err := store.MarkFileFailed(context.Background(), FailedFileInput{
		LibraryID:         "library-1",
		Path:              "/downloads/Broken.2020.mkv",
		DetectedMediaType: "movie",
		ParsedTitle:       "Broken",
		ParsedYear:        2020,
		Error:             "library id is required",
	})
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != FileStatusFailed || failed.FailureError == "" || failed.FailedAt == nil {
		t.Fatalf("expected failed file with failure details, got %#v", failed)
	}

	recovered, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID:         "library-1",
		Path:              "/downloads/Broken.2020.mkv",
		DetectedMediaType: "movie",
		ParsedTitle:       "Broken",
		ParsedYear:        2020,
	})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.ID != failed.ID {
		t.Fatalf("expected recovered file to keep id %q, got %q", failed.ID, recovered.ID)
	}
	if recovered.Status != FileStatusAvailable || recovered.FailureError != "" || recovered.FailedAt != nil {
		t.Fatalf("expected successful upsert to clear failure details, got %#v", recovered)
	}
}

func TestMemoryStoreListFilesFiltersFailures(t *testing.T) {
	store := NewMemoryStore()
	firstTime := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Hour)
	store.now = func() time.Time { return firstTime }
	if _, err := store.MarkFileFailed(context.Background(), FailedFileInput{
		LibraryID:         "library-1",
		Path:              "/downloads/retry/Arrival.2016.mkv",
		DetectedMediaType: "movie",
		Error:             "transient parser failure",
	}); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return secondTime }
	if _, err := store.MarkFileFailed(context.Background(), FailedFileInput{
		LibraryID:         "library-1",
		Path:              "/downloads/skip/Show.S01E01.mkv",
		DetectedMediaType: "tv",
		Error:             "permanent scanner failure",
	}); err != nil {
		t.Fatal(err)
	}

	files, err := store.ListFiles(context.Background(), FileQuery{
		LibraryID:         "library-1",
		Status:            FileStatusFailed,
		PathPrefix:        "/downloads/retry",
		DetectedMediaType: "movie",
		FailureContains:   "parser",
		FailedBefore:      &secondTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "/downloads/retry/Arrival.2016.mkv" {
		t.Fatalf("expected filtered retry file, got %#v", files)
	}
}

func TestMemoryStoreUpdateFilePath(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	file, err := store.UpsertFile(context.Background(), FileInput{
		LibraryID: "library-1",
		Path:      "/downloads/Inception.tmp",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, ok, err := store.UpdateFilePath(context.Background(), file.ID, "/media/Inception.mkv")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected file to be updated")
	}
	if updated.Path != "/media/Inception.mkv" || updated.FileName != "Inception.mkv" || updated.Extension != ".mkv" {
		t.Fatalf("unexpected updated file: %#v", updated)
	}
	if _, ok, err := store.GetFileByPath(context.Background(), "/downloads/Inception.tmp"); err != nil || ok {
		t.Fatalf("expected old path lookup to miss, ok=%v err=%v", ok, err)
	}
	if found, ok, err := store.GetFileByPath(context.Background(), "/media/Inception.mkv"); err != nil || !ok || found.ID != file.ID {
		t.Fatalf("expected new path lookup to find file, found=%#v ok=%v err=%v", found, ok, err)
	}
}
