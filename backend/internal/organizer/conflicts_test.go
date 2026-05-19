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
