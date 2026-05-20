package organizer

import (
	"fmt"
	"strings"
	"time"
)

type ConflictFilter struct {
	ActionIDs        []string
	ActionType       ActionMode
	ConflictReason   string
	SourcePathPrefix string
	TargetPathPrefix string
}

type ConflictOperation string

const (
	ConflictOperationSkip             ConflictOperation = "skip"
	ConflictOperationRename           ConflictOperation = "rename"
	ConflictOperationConfirmOverwrite ConflictOperation = "confirm-overwrite"
)

func (f ConflictFilter) Matches(action Action) bool {
	if len(f.ActionIDs) > 0 && !containsString(f.ActionIDs, action.ID) {
		return false
	}
	if f.ActionType != "" && action.ActionType != f.ActionType {
		return false
	}
	if f.ConflictReason != "" && action.ConflictReason != f.ConflictReason {
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

func SkipConflicts(plan Plan, now time.Time, filter ConflictFilter) (Plan, int) {
	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict || !filter.Matches(action) {
			continue
		}
		action.Status = ActionSkipped
		action.Error = ""
		plan.Actions[index] = action
		changed++
	}
	if changed > 0 {
		plan.Status = PlanReady
		plan.Summary = SummarizeActions(plan.Actions)
		plan.UpdatedAt = now.UTC()
	}
	return plan, changed
}

func RenameConflicts(plan Plan, now time.Time, targetExists func(string) bool, filter ConflictFilter) (Plan, int) {
	planner := Planner{TargetExists: targetExists}
	seenTargets := make(map[string]string, len(plan.Actions))
	for _, action := range plan.Actions {
		if action.Status != ActionConflict || !filter.Matches(action) {
			seenTargets[action.TargetPath] = action.SourcePath
		}
	}

	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict || !filter.Matches(action) {
			continue
		}
		action.TargetPath = planner.nextAvailableTarget(action.TargetPath, seenTargets)
		action.Status = ActionPending
		action.ConflictReason = ""
		action.Error = ""
		seenTargets[action.TargetPath] = action.SourcePath
		plan.Actions[index] = action
		changed++
	}
	if changed > 0 {
		plan.Status = PlanReady
		plan.Summary = SummarizeActions(plan.Actions)
		plan.UpdatedAt = now.UTC()
	}
	return plan, changed
}

func ConfirmOverwriteConflicts(plan Plan, now time.Time, filter ConflictFilter) (Plan, int) {
	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict || action.ConflictReason != ConflictReasonTargetPathExists || !filter.Matches(action) {
			continue
		}
		action.Status = ActionPending
		action.ConflictReason = ConflictReasonOverwriteConfirmed
		action.Error = ""
		action.ExecutedAt = nil
		plan.Actions[index] = action
		changed++
	}
	if changed > 0 {
		plan.Status = PlanReady
		plan.Summary = SummarizeActions(plan.Actions)
		plan.UpdatedAt = now.UTC()
	}
	return plan, changed
}

func PreviewConflicts(plan Plan, operation ConflictOperation, filter ConflictFilter) ([]Action, error) {
	if operation == "" {
		operation = ConflictOperationSkip
	}
	switch operation {
	case ConflictOperationSkip, ConflictOperationRename, ConflictOperationConfirmOverwrite:
	default:
		return nil, errUnsupportedConflictOperation(operation)
	}
	actions := make([]Action, 0)
	for _, action := range plan.Actions {
		if action.Status != ActionConflict || !filter.Matches(action) {
			continue
		}
		if operation == ConflictOperationConfirmOverwrite && action.ConflictReason != ConflictReasonTargetPathExists {
			continue
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func errUnsupportedConflictOperation(operation ConflictOperation) error {
	return fmt.Errorf("unsupported conflict operation %q", operation)
}
