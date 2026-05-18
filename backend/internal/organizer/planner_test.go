package organizer

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPlannerBuildMovieMultiVersionSameFolder(t *testing.T) {
	planner := Planner{Now: fixedNow}
	plan, err := planner.Build(PlanRequest{
		Media: MediaInfo{
			ID:        "media-1",
			LibraryID: "library-1",
			MediaType: MediaTypeMovie,
			Title:     "Inception",
			Year:      2010,
		},
		Versions: []VersionInfo{
			{ID: "v-4k", Resolution: "2160p", Source: "bluray"},
			{ID: "v-hd", Resolution: "1080p", Source: "web-dl"},
		},
		Files: []FileInfo{
			{ID: "file-1", VersionID: "v-4k", Path: "/downloads/Inception.2010.2160p.mkv"},
			{ID: "file-2", VersionID: "v-hd", Path: "/downloads/Inception.2010.1080p.mkv"},
		},
		Rule: Rule{
			LibraryID:  "library-1",
			TargetRoot: "/library/movies",
			ActionMode: ActionMove,
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	wantFolder := filepath.Join("/library/movies", "Inception (2010)")
	assertTarget(t, plan.Actions[0], filepath.Join(wantFolder, "Inception (2010) - 2160p bluray.mkv"))
	assertTarget(t, plan.Actions[1], filepath.Join(wantFolder, "Inception (2010) - 1080p web-dl.mkv"))
	if filepath.Dir(plan.Actions[0].TargetPath) != filepath.Dir(plan.Actions[1].TargetPath) {
		t.Fatalf("expected versions in same folder, got %q and %q", plan.Actions[0].TargetPath, plan.Actions[1].TargetPath)
	}
	assertSummary(t, plan.Summary, 2, 2, 0)
}

func TestPlannerBuildTVSeasonFolder(t *testing.T) {
	planner := Planner{Now: fixedNow}
	plan, err := planner.Build(PlanRequest{
		Media: MediaInfo{
			ID:        "media-tv",
			LibraryID: "library-tv",
			MediaType: MediaTypeTV,
			Title:     "Severance",
			Year:      2022,
		},
		Versions: []VersionInfo{
			{ID: "v-1", Resolution: "1080p", Source: "web-dl"},
		},
		Files: []FileInfo{
			{ID: "file-1", VersionID: "v-1", Path: "/downloads/Severance.S02E03.mkv", Season: 2, Episode: 3},
		},
		Rule: Rule{
			LibraryID:  "library-tv",
			TargetRoot: "/library/tv",
			ActionMode: ActionMove,
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertTarget(t, plan.Actions[0], filepath.Join("/library/tv", "Severance (2022)", "Season 02", "Severance - S02E03 - 1080p web-dl.mkv"))
	assertSummary(t, plan.Summary, 1, 1, 0)
}

func TestPlannerBuildAVNumberFolder(t *testing.T) {
	planner := Planner{Now: fixedNow}
	plan, err := planner.Build(PlanRequest{
		Media: MediaInfo{
			ID:        "media-av",
			LibraryID: "library-av",
			MediaType: MediaTypeAV,
			Title:     "Sample Title",
			Number:    "ABP-123",
		},
		Versions: []VersionInfo{
			{ID: "v-1", Resolution: "1080p", Source: "web-dl"},
		},
		Files: []FileInfo{
			{ID: "file-1", VersionID: "v-1", Path: "/downloads/ABP-123.mp4"},
		},
		Rule: Rule{
			LibraryID:  "library-av",
			TargetRoot: "/library/av",
			ActionMode: ActionMove,
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertTarget(t, plan.Actions[0], filepath.Join("/library/av", "ABP-123 Sample Title", "ABP-123 - 1080p web-dl.mp4"))
	assertSummary(t, plan.Summary, 1, 1, 0)
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
}

func assertTarget(t *testing.T, action Action, want string) {
	t.Helper()
	if action.TargetPath != want {
		t.Fatalf("TargetPath = %q, want %q", action.TargetPath, want)
	}
	if action.Status != ActionPending {
		t.Fatalf("Status = %q, want %q", action.Status, ActionPending)
	}
}

func assertSummary(t *testing.T, summary Summary, total int, moves int, conflicts int) {
	t.Helper()
	if summary.TotalActions != total || summary.MoveCount != moves || summary.ConflictCount != conflicts {
		t.Fatalf("Summary = %+v, want total=%d moves=%d conflicts=%d", summary, total, moves, conflicts)
	}
}
