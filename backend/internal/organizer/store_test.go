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

func TestMemoryStoreRuleCRUD(t *testing.T) {
	store := NewMemoryStore()
	rule, err := store.CreateRule(context.Background(), RuleInput{
		Name:       "Movies",
		LibraryID:  "library-1",
		MediaType:  "movie",
		TargetRoot: "/library/movies",
		Enabled:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.ActionMode != ActionMove {
		t.Fatalf("expected default move action, got %s", rule.ActionMode)
	}
	if rule.ConflictPolicy != ConflictSkip {
		t.Fatalf("expected default skip conflict policy, got %s", rule.ConflictPolicy)
	}

	targetRoot := "/library/films"
	updated, ok, err := store.UpdateRule(context.Background(), rule.ID, RuleUpdate{TargetRoot: &targetRoot})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected rule update")
	}
	if updated.TargetRoot != targetRoot {
		t.Fatalf("expected target root update, got %q", updated.TargetRoot)
	}

	rules, err := store.ListRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected one rule, got %d", len(rules))
	}
}
