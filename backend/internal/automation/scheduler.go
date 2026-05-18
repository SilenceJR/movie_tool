package automation

import (
	"context"
	"fmt"
	"time"

	"movie-tool/backend/internal/task"
)

type TaskQueue interface {
	Enqueue(task.Type, string) task.Task
}

type Scheduler struct {
	Queue TaskQueue
	Now   func() time.Time
}

type TickResult struct {
	QueuedTasks []task.Task
	Skipped     int
	Errors      []error
}

func (s Scheduler) Tick(ctx context.Context, automations []Automation) TickResult {
	now := s.now()
	result := TickResult{}

	for _, automation := range automations {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ctx.Err())
			return result
		default:
		}

		if !automation.Enabled {
			result.Skipped++
			continue
		}
		if automation.NextRunAt != nil && automation.NextRunAt.After(now) {
			result.Skipped++
			continue
		}

		taskType, err := taskTypeForAutomation(automation.Type)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}
		if s.Queue == nil {
			result.Errors = append(result.Errors, fmt.Errorf("task queue is nil"))
			continue
		}

		queued := s.Queue.Enqueue(taskType, "automation: "+automation.Name)
		result.QueuedTasks = append(result.QueuedTasks, queued)
	}

	return result
}

func (s Scheduler) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func taskTypeForAutomation(automationType Type) (task.Type, error) {
	switch automationType {
	case TypeScanLibrary:
		return task.TypeLibraryScan, nil
	case TypeScrapePending:
		return task.TypeScrapeMedia, nil
	case TypeTranslateMissing:
		return task.TypeTranslateMetadata, nil
	case TypeOrganizeFiles:
		return task.TypeOrganizeFiles, nil
	case TypeGenerateNFO:
		return task.TypeGenerateNFO, nil
	case TypeGenerateSTRM:
		return task.TypeGenerateSTRM, nil
	case TypeRefreshServer:
		return task.TypeRefreshServer, nil
	case TypeCleanupMissing:
		return task.TypeCleanupMissing, nil
	default:
		return "", fmt.Errorf("unsupported automation type %q", automationType)
	}
}
