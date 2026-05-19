package organizer

import (
	"context"
	"testing"
)

func TestMemoryStorePlan(t *testing.T) {
	store := NewMemoryStore()

	plan, err := store.SavePlan(context.Background(), Plan{
		ID:        "plan-1",
		LibraryID: "library-1",
		Status:    PlanReady,
		DryRun:    true,
		Actions: []Action{
			{ID: "action-1", PlanID: "plan-1", SourcePath: "/a.mkv", TargetPath: "/b.mkv", Status: ActionPending},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("expected saved action, got %d", len(plan.Actions))
	}

	found, ok, err := store.GetPlan(context.Background(), "plan-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected plan")
	}
	if len(found.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(found.Actions))
	}

	actions, err := store.ListActions(context.Background(), "plan-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
}
