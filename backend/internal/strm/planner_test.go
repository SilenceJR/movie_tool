package strm

import "testing"

func TestPlannerBuild(t *testing.T) {
	rule := Rule{
		ID:           "rule-1",
		Name:         "NAS",
		SourcePrefix: "/mnt/media",
		TargetPrefix: "http://nas.local/media",
		OutputPath:   "/strm",
		Enabled:      true,
	}
	plan, err := NewPlanner().Build(rule, GenerateRequest{
		LibraryID: "library-1",
		Files: []FileInfo{
			{ID: "file-1", Path: "/mnt/media/Movies/Inception.mkv"},
			{ID: "file-2", Path: "/other/file.mkv"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Count != 2 {
		t.Fatalf("expected 2 entries, got %d", plan.Count)
	}
	if plan.Entries[0].TargetURL != "http://nas.local/media/Movies/Inception.mkv" {
		t.Fatalf("unexpected target url %q", plan.Entries[0].TargetURL)
	}
	if plan.Entries[0].Content != "http://nas.local/media/Movies/Inception.mkv\n" {
		t.Fatalf("unexpected content %q", plan.Entries[0].Content)
	}
	if plan.Entries[1].Status != "skipped" {
		t.Fatalf("expected skipped outside prefix, got %q", plan.Entries[1].Status)
	}
}

func TestPlannerValidate(t *testing.T) {
	result := NewPlanner().Validate(RuleInput{
		Name:         "NAS",
		SourcePrefix: "/mnt/media",
		TargetPrefix: "http://nas.local/media",
		OutputPath:   "/strm",
	}, "/mnt/media/Movies/Inception.mkv")
	if !result.Valid {
		t.Fatalf("expected valid mapping, got %s", result.Error)
	}
	if result.OutputPath != "/strm/Movies/Inception.strm" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
}
