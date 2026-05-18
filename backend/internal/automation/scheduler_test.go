package automation

import (
	"context"
	"testing"
	"time"

	"movie-tool/backend/internal/task"
)

func TestSchedulerQueuesDueAutomation(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	queue := task.NewQueue()
	scheduler := Scheduler{
		Queue: queue,
		Now: func() time.Time {
			return now
		},
	}
	due := now.Add(-time.Minute)

	result := scheduler.Tick(context.Background(), []Automation{
		{
			Name:      "Scan",
			Type:      TypeScanLibrary,
			Enabled:   true,
			NextRunAt: &due,
		},
	})

	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.QueuedTasks) != 1 {
		t.Fatalf("expected 1 queued task, got %d", len(result.QueuedTasks))
	}
	if result.QueuedTasks[0].Type != task.TypeLibraryScan {
		t.Fatalf("expected library scan task, got %s", result.QueuedTasks[0].Type)
	}
}

func TestSchedulerSkipsFutureAutomation(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	queue := task.NewQueue()
	scheduler := Scheduler{
		Queue: queue,
		Now: func() time.Time {
			return now
		},
	}
	future := now.Add(time.Hour)

	result := scheduler.Tick(context.Background(), []Automation{
		{
			Name:      "Scan",
			Type:      TypeScanLibrary,
			Enabled:   true,
			NextRunAt: &future,
		},
	})

	if len(result.QueuedTasks) != 0 {
		t.Fatalf("expected no queued tasks, got %d", len(result.QueuedTasks))
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped automation, got %d", result.Skipped)
	}
}
