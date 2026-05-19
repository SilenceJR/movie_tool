package organizer

import (
	"strings"
	"time"
)

const mediaFilePathUpdateErrorPrefix = "update media file path:"

func PrepareRetry(plan Plan, now time.Time) (Plan, int) {
	retryable := 0
	for index, action := range plan.Actions {
		if action.Status != ActionFailed {
			continue
		}
		retryable++
		if strings.HasPrefix(action.Error, mediaFilePathUpdateErrorPrefix) {
			action.Status = ActionSucceeded
		} else {
			action.Status = ActionPending
			action.ExecutedAt = nil
		}
		action.Error = ""
		plan.Actions[index] = action
	}
	if retryable > 0 {
		plan.Status = PlanReady
		plan.Summary = SummarizeActions(plan.Actions)
		plan.UpdatedAt = now.UTC()
	}
	return plan, retryable
}
