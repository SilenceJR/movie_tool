package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWalkFindsMovieFiles(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "Movies", "Inception.2010.2160p.BluRay.REMUX.HEVC.mkv")
	touch(t, root, "Movies", "poster.jpg")
	touch(t, root, "Movies", "Inception.2010.nfo")

	files, err := Walk(root, LibraryInfo{
		ID:        "library-movie",
		Name:      "Movies",
		MediaType: "movie",
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 media file, got %d", len(files))
	}
	if files[0].Title != "Inception" || files[0].Year != 2010 {
		t.Fatalf("unexpected parsed movie: %+v", files[0])
	}
	if files[0].LibraryID != "library-movie" || files[0].MediaType != "movie" {
		t.Fatalf("expected library metadata on parsed file, got %+v", files[0])
	}
}

func TestWalkFindsTVEpisodes(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "Show Name", "Season 02", "Show.Name.S02E03.1080p.WEB-DL.H264.mkv")
	touch(t, root, "Show Name", "Season 02", "Show.Name.S02E03.zh-CN.srt")

	files, err := Walk(root, LibraryInfo{ID: "library-tv", MediaType: "tv"})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 media file, got %d", len(files))
	}
	if files[0].Title != "Show Name" || files[0].Season != 2 || files[0].Episode != 3 {
		t.Fatalf("unexpected parsed episode: %+v", files[0])
	}
}

func TestWalkFindsAVFiles(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "ABC-123-C.mp4")
	touch(t, root, "cover.png")

	files, err := Walk(root, LibraryInfo{ID: "library-av", MediaType: "av"})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 media file, got %d", len(files))
	}
	if files[0].Number != "ABC-123" || files[0].Title != "ABC-123" {
		t.Fatalf("unexpected parsed AV file: %+v", files[0])
	}
}

func TestWalkIgnoresHiddenAndNonMediaFiles(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "Movie.2020.mkv")
	touch(t, root, ".hidden.mp4")
	touch(t, root, ".hidden-dir", "Hidden.2020.mkv")
	touch(t, root, "notes.txt")
	touch(t, root, "Movie.2020.ass")

	files, err := Walk(root, LibraryInfo{ID: "library-mixed", MediaType: "movie"})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected only visible media file, got %d: %+v", len(files), files)
	}
	if files[0].FileName != "Movie.2020.mkv" {
		t.Fatalf("expected visible movie file, got %q", files[0].FileName)
	}
}

func TestWalkUsesLibraryPathWhenRootIsEmpty(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "FC2-PPV-1234567.mp4")

	files, err := NewScanner().Walk(ScanRequest{
		Library: LibraryInfo{ID: "library-path", Path: root, MediaType: "av"},
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 media file, got %d", len(files))
	}
	if files[0].Number != "FC2-PPV-1234567" {
		t.Fatalf("unexpected parsed file: %+v", files[0])
	}
}

func TestWalkSkipsRecentlyModifiedFilesWhenStableAgeIsSet(t *testing.T) {
	root := t.TempDir()
	oldPath := touch(t, root, "Stable.2020.mkv")
	recentPath := touch(t, root, "Writing.2024.mkv")
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, now.Add(-10*time.Minute), now.Add(-10*time.Minute)); err != nil {
		t.Fatalf("set old mtime: %v", err)
	}
	if err := os.Chtimes(recentPath, now.Add(-30*time.Second), now.Add(-30*time.Second)); err != nil {
		t.Fatalf("set recent mtime: %v", err)
	}

	files, err := NewScanner().Walk(ScanRequest{
		Root:           root,
		Library:        LibraryInfo{ID: "library-stable", MediaType: "movie"},
		MinModifiedAge: 2 * time.Minute,
		Now:            func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if len(files) != 1 || files[0].FileName != "Stable.2020.mkv" {
		t.Fatalf("expected only stable file, got %+v", files)
	}
}

func touch(t *testing.T, root string, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{root}, parts...)...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	return path
}
