package organizer

import "time"

func SkipConflicts(plan Plan, now time.Time) (Plan, int) {
	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict {
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

func RenameConflicts(plan Plan, now time.Time, targetExists func(string) bool) (Plan, int) {
	planner := Planner{TargetExists: targetExists}
	seenTargets := make(map[string]string, len(plan.Actions))
	for _, action := range plan.Actions {
		if action.Status != ActionConflict {
			seenTargets[action.TargetPath] = action.SourcePath
		}
	}

	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict {
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

func ConfirmOverwriteConflicts(plan Plan, now time.Time) (Plan, int) {
	changed := 0
	for index, action := range plan.Actions {
		if action.Status != ActionConflict || action.ConflictReason != ConflictReasonTargetPathExists {
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
