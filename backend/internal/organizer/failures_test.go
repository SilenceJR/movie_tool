package organizer

import "testing"

func TestSkipFailedActionsMarksFilteredFailuresSkipped(t *testing.T) {
	plan, changed := SkipFailedActions(Plan{
		ID:     "plan-1",
		Status: PlanFailed,
		Actions: []Action{
			{ID: "a1", ActionType: ActionCopy, SourcePath: "/downloads/a.mkv", TargetPath: "/library/a.mkv", Status: ActionFailed, Error: "permission denied"},
			{ID: "a2", ActionType: ActionCopy, SourcePath: "/downloads/b.mkv", TargetPath: "/library/b.mkv", Status: ActionFailed, Error: "disk full"},
			{ID: "a3", ActionType: ActionMove, SourcePath: "/downloads/c.mkv", TargetPath: "/library/c.mkv", Status: ActionSucceeded},
		},
	}, fixedNow(), FailureFilter{ErrorContains: "permission"})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Actions[0].Status != ActionSkipped || plan.Actions[0].Error != "" || plan.Actions[0].ExecutedAt != nil {
		t.Fatalf("expected first failed action skipped and cleared, got %+v", plan.Actions[0])
	}
	if plan.Actions[1].Status != ActionFailed {
		t.Fatalf("expected unfiltered failed action to remain failed, got %+v", plan.Actions[1])
	}
	if plan.Status != PlanFailed || plan.Summary.SkipCount != 1 {
		t.Fatalf("unexpected repaired plan state: status=%s summary=%+v", plan.Status, plan.Summary)
	}
}

func TestSkipFailedActionsMarksPlanSucceededWhenNoFailuresRemain(t *testing.T) {
	plan, changed := SkipFailedActions(Plan{
		ID:     "plan-1",
		Status: PlanFailed,
		Actions: []Action{
			{ID: "a1", ActionType: ActionCopy, Status: ActionFailed, Error: "already handled"},
			{ID: "a2", ActionType: ActionMove, Status: ActionSucceeded},
		},
	}, fixedNow(), FailureFilter{ActionIDs: []string{"a1"}})

	if changed != 1 {
		t.Fatalf("expected one changed action, got %d", changed)
	}
	if plan.Status != PlanSucceeded {
		t.Fatalf("expected repaired plan to become succeeded, got %s", plan.Status)
	}
	if plan.Summary.SkipCount != 1 || plan.Summary.TotalActions != 2 {
		t.Fatalf("unexpected summary after repair: %+v", plan.Summary)
	}
}

func TestPreviewFailedActionsAppliesFilter(t *testing.T) {
	actions := PreviewFailedActions(Plan{
		ID:     "plan-1",
		Status: PlanFailed,
		Actions: []Action{
			{ID: "a1", ActionType: ActionCopy, TargetPath: "/library/a.mkv", Status: ActionFailed, Error: "permission denied"},
			{ID: "a2", ActionType: ActionMove, TargetPath: "/library/b.mkv", Status: ActionFailed, Error: "disk full"},
			{ID: "a3", ActionType: ActionCopy, TargetPath: "/library/c.mkv", Status: ActionSucceeded},
		},
	}, FailureFilter{ActionType: ActionCopy, ErrorContains: "permission"})

	if len(actions) != 1 || actions[0].ID != "a1" {
		t.Fatalf("expected filtered failed action preview, got %+v", actions)
	}
}
