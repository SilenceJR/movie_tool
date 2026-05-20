package organizer

import (
	"strings"
	"time"
)

type FailureFilter struct {
	ActionIDs        []string
	ActionType       ActionMode
	ErrorContains    string
	SourcePathPrefix string
	TargetPathPrefix string
}

func (f FailureFilter) Matches(action Action) bool {
	if len(f.ActionIDs) > 0 && !containsString(f.ActionIDs, action.ID) {
		return false
	}
	if f.ActionType != "" && action.ActionType != f.ActionType {
		return false
	}
	if f.ErrorContains != "" && !strings.Contains(strings.ToLower(action.Error), strings.ToLower(f.ErrorContains)) {
		return false
	}
	if f.SourcePathPrefix != "" && !strings.HasPrefix(action.SourcePath, f.SourcePathPrefix) {
		return false
	}
	if f.TargetPathPrefix != "" && !strings.HasPrefix(action.TargetPath, f.TargetPathPrefix) {
		return false
	}
	return true
}

func SkipFailedActions(plan Plan, now time.Time, filter FailureFilter) (Plan, int) {
	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionFailed || !filter.Matches(action) {
			continue
		}
		action.Status = ActionSkipped
		action.Error = ""
		action.ExecutedAt = nil
		plan.Actions[index] = action
		changed++
	}
	if changed > 0 {
		plan.Summary = SummarizeActions(plan.Actions)
		plan.Status = statusAfterManualFailureRepair(plan.Actions)
		plan.UpdatedAt = now.UTC()
	}
	return plan, changed
}

func PreviewFailedActions(plan Plan, filter FailureFilter) []Action {
	actions := make([]Action, 0)
	for _, action := range plan.Actions {
		if action.Status == ActionFailed && filter.Matches(action) {
			actions = append(actions, action)
		}
	}
	return actions
}

func statusAfterManualFailureRepair(actions []Action) PlanStatus {
	for _, action := range actions {
		switch action.Status {
		case ActionFailed:
			return PlanFailed
		case ActionPending:
			return PlanRunning
		case ActionConflict:
			return PlanReady
		}
	}
	return PlanSucceeded
}
