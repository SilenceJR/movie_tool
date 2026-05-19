package organizer

import "testing"

func TestPrepareRetryResetsFailedFileActions(t *testing.T) {
	executedAt := fixedNow()
	plan, retryable := PrepareRetry(Plan{
		ID:     "plan-1",
		Status: PlanFailed,
		Actions: []Action{
			{ID: "a1", Status: ActionFailed, Error: "copy failed", ExecutedAt: &executedAt},
			{ID: "a2", Status: ActionSucceeded, ExecutedAt: &executedAt},
		},
	}, fixedNow())

	if retryable != 1 {
		t.Fatalf("expected one retryable action, got %d", retryable)
	}
	if plan.Status != PlanReady {
		t.Fatalf("expected ready plan, got %s", plan.Status)
	}
	if plan.Actions[0].Status != ActionPending || plan.Actions[0].Error != "" || plan.Actions[0].ExecutedAt != nil {
		t.Fatalf("expected failed action reset to pending, got %#v", plan.Actions[0])
	}
	if plan.Actions[1].Status != ActionSucceeded {
		t.Fatalf("expected succeeded action preserved, got %#v", plan.Actions[1])
	}
}

func TestPrepareRetryKeepsMovedFileActionForPathUpdateRetry(t *testing.T) {
	executedAt := fixedNow()
	plan, retryable := PrepareRetry(Plan{
		ID:     "plan-1",
		Status: PlanFailed,
		Actions: []Action{
			{ID: "a1", Status: ActionFailed, Error: mediaFilePathUpdateErrorPrefix + " database busy", ExecutedAt: &executedAt},
		},
	}, fixedNow())

	if retryable != 1 {
		t.Fatalf("expected one retryable action, got %d", retryable)
	}
	if plan.Actions[0].Status != ActionSucceeded || plan.Actions[0].Error != "" || plan.Actions[0].ExecutedAt == nil {
		t.Fatalf("expected path-update failure to retry only database update, got %#v", plan.Actions[0])
	}
}
