package organizer

import "testing"

func TestSkipConflictsMarksConflictActionsSkipped(t *testing.T) {
	plan, changed := SkipConflicts(Plan{
		ID:     "plan-1",
		Status: PlanReady,
		Actions: []Action{
			{ID: "a1", Status: ActionConflict, ConflictReason: "target exists"},
			{ID: "a2", Status: ActionPending},
		},
	}, fixedNow())

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Actions[0].Status != ActionSkipped || plan.Actions[1].Status != ActionPending {
		t.Fatalf("unexpected actions after skip: %+v", plan.Actions)
	}
	if plan.Summary.SkipCount != 1 || plan.Summary.TotalActions != 2 {
		t.Fatalf("unexpected summary after skip: %+v", plan.Summary)
	}
	if plan.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at to be set")
	}
}

func TestRenameConflictsMovesConflictActionsToNextAvailableTarget(t *testing.T) {
	plan, changed := RenameConflicts(Plan{
		ID:     "plan-1",
		Status: PlanReady,
		Actions: []Action{
			{ID: "a1", SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionConflict, ConflictReason: "target exists"},
			{ID: "a2", SourcePath: "/downloads/b.mkv", TargetPath: "/library/b (1).mkv", Status: ActionPending},
		},
	}, fixedNow(), func(path string) bool {
		return path == "/library/a (1).mkv"
	})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	action := plan.Actions[0]
	if action.Status != ActionPending || action.TargetPath != "/library/a (2).mkv" || action.ConflictReason != "" {
		t.Fatalf("unexpected renamed conflict action: %+v", action)
	}
	if plan.Summary.TotalActions != 2 || plan.Summary.ConflictCount != 0 {
		t.Fatalf("unexpected summary after rename: %+v", plan.Summary)
	}
}
