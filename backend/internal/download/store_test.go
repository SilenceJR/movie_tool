package download

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreListWatchEnabledFiltersAndSorts(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		now = now.Add(time.Second)
		return now
	}

	ctx := context.Background()
	first := createDirectory(t, ctx, store, DirectoryInput{
		Name:         "first",
		Path:         "/downloads/first",
		LibraryID:    "library_movies",
		ActionMode:   "hardlink",
		Enabled:      true,
		WatchEnabled: true,
	})
	_ = createDirectory(t, ctx, store, DirectoryInput{
		Name:         "watch disabled",
		Path:         "/downloads/watch-disabled",
		LibraryID:    "library_movies",
		ActionMode:   "hardlink",
		Enabled:      true,
		WatchEnabled: false,
	})
	_ = createDirectory(t, ctx, store, DirectoryInput{
		Name:         "disabled",
		Path:         "/downloads/disabled",
		LibraryID:    "library_movies",
		ActionMode:   "hardlink",
		Enabled:      false,
		WatchEnabled: true,
	})
	second := createDirectory(t, ctx, store, DirectoryInput{
		Name:         "second",
		Path:         "/downloads/second",
		LibraryID:    "library_tv",
		ActionMode:   "copy",
		Enabled:      true,
		WatchEnabled: true,
	})

	directories, err := store.ListWatchEnabled(ctx)
	if err != nil {
		t.Fatalf("ListWatchEnabled returned error: %v", err)
	}

	if len(directories) != 2 {
		t.Fatalf("expected 2 watch-enabled directories, got %d: %#v", len(directories), directories)
	}
	if directories[0].ID != first.ID || directories[1].ID != second.ID {
		t.Fatalf("expected watch-enabled directories sorted by creation time, got %q then %q", directories[0].ID, directories[1].ID)
	}
}

func createDirectory(t *testing.T, ctx context.Context, store *MemoryStore, input DirectoryInput) Directory {
	t.Helper()
	directory, err := store.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	return directory
}
