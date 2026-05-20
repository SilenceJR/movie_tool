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
	}, fixedNow(), ConflictFilter{})

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
	}, ConflictFilter{})

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

func TestConfirmOverwriteConflictsMarksExistingTargetConflictsPending(t *testing.T) {
	plan, changed := ConfirmOverwriteConflicts(Plan{
		ID:     "plan-1",
		Status: PlanReady,
		Actions: []Action{
			{ID: "a1", SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionConflict, ConflictReason: ConflictReasonTargetPathExists},
			{ID: "a2", SourcePath: "/downloads/b.mkv", TargetPath: "/library/a.mkv", Status: ActionConflict, ConflictReason: ConflictReasonDuplicateTargetPath},
		},
	}, fixedNow(), ConflictFilter{})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Actions[0].Status != ActionPending || plan.Actions[0].ConflictReason != ConflictReasonOverwriteConfirmed {
		t.Fatalf("expected existing target conflict to be confirmed, got %+v", plan.Actions[0])
	}
	if plan.Actions[1].Status != ActionConflict || plan.Actions[1].ConflictReason != ConflictReasonDuplicateTargetPath {
		t.Fatalf("expected duplicate target conflict to stay unchanged, got %+v", plan.Actions[1])
	}
	if plan.Summary.TotalActions != 2 || plan.Summary.ConflictCount != 1 {
		t.Fatalf("unexpected summary after confirm overwrite: %+v", plan.Summary)
	}
}

func TestSkipConflictsAppliesFilter(t *testing.T) {
	plan, changed := SkipConflicts(Plan{
		ID:     "plan-1",
		Status: PlanReady,
		Actions: []Action{
			{ID: "a1", ActionType: ActionCopy, TargetPath: "/library/movie/a.mkv", Status: ActionConflict, ConflictReason: ConflictReasonTargetPathExists},
			{ID: "a2", ActionType: ActionCopy, TargetPath: "/library/show/a.mkv", Status: ActionConflict, ConflictReason: ConflictReasonTargetPathExists},
		},
	}, fixedNow(), ConflictFilter{TargetPathPrefix: "/library/movie"})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Actions[0].Status != ActionSkipped || plan.Actions[1].Status != ActionConflict {
		t.Fatalf("unexpected filtered skip actions: %+v", plan.Actions)
	}
}

func TestRenameConflictsFilterKeepsUnselectedConflictTargetsReserved(t *testing.T) {
	plan, changed := RenameConflicts(Plan{
		ID:     "plan-1",
		Status: PlanReady,
		Actions: []Action{
			{ID: "a1", SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionConflict, ConflictReason: ConflictReasonTargetPathExists},
			{ID: "a2", SourcePath: "/downloads/b.mkv", TargetPath: "/library/a (1).mkv", Status: ActionConflict, ConflictReason: ConflictReasonTargetPathExists},
		},
	}, fixedNow(), func(string) bool { return false }, ConflictFilter{ActionIDs: []string{"a1"}})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Actions[0].TargetPath != "/library/a (2).mkv" {
		t.Fatalf("expected filtered rename to avoid unselected conflict target, got %+v", plan.Actions[0])
	}
	if plan.Actions[1].Status != ActionConflict || plan.Actions[1].TargetPath != "/library/a (1).mkv" {
		t.Fatalf("expected unselected conflict to stay unchanged, got %+v", plan.Actions[1])
	}
}
