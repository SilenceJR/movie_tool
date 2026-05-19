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
