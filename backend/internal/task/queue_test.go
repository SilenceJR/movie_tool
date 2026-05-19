package task

import (
	"context"
	"errors"
	"testing"
)

func TestQueueRunSuccess(t *testing.T) {
	queue := NewQueue()
	queue.Register(TypeLibraryScan, func(context.Context, Task) error {
		return nil
	})

	created := queue.Enqueue(TypeLibraryScan, "scan")
	done := queue.Run(context.Background(), created.ID)

	if done.Status != StatusSucceeded {
		t.Fatalf("expected succeeded, got %s", done.Status)
	}
	if done.Progress != 100 {
		t.Fatalf("expected progress 100, got %d", done.Progress)
	}
	logs := queue.Logs(created.ID)
	if len(logs) != 3 {
		t.Fatalf("expected queued/start/success logs, got %d", len(logs))
	}
	if logs[2].Message != "task succeeded" {
		t.Fatalf("expected success log, got %q", logs[2].Message)
	}
}

func TestQueueRunFailure(t *testing.T) {
	queue := NewQueue()
	queue.Register(TypeLibraryScan, func(context.Context, Task) error {
		return errors.New("boom")
	})

	created := queue.Enqueue(TypeLibraryScan, "scan")
	done := queue.Run(context.Background(), created.ID)

	if done.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", done.Status)
	}
	if done.Error != "boom" {
		t.Fatalf("expected boom error, got %q", done.Error)
	}
	logs := queue.Logs(created.ID)
	if len(logs) != 3 {
		t.Fatalf("expected queued/start/error logs, got %d", len(logs))
	}
	if logs[2].Level != LogLevelError {
		t.Fatalf("expected error log level, got %s", logs[2].Level)
	}
}

func TestQueueMissingHandler(t *testing.T) {
	queue := NewQueue()
	created := queue.Enqueue(TypeLibraryScan, "scan")
	done := queue.Run(context.Background(), created.ID)

	if done.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", done.Status)
	}
}

func TestQueueCancel(t *testing.T) {
	queue := NewQueue()
	created := queue.Enqueue(TypeLibraryScan, "scan")

	canceled, ok := queue.Cancel(created.ID)
	if !ok {
		t.Fatal("expected cancel to find task")
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}

	canceledTasks := queue.ListByQuery(Query{Status: StatusCanceled})
	if len(canceledTasks) != 1 {
		t.Fatalf("expected 1 canceled task, got %d", len(canceledTasks))
	}
	logs := queue.Logs(created.ID)
	if len(logs) != 2 {
		t.Fatalf("expected queued/canceled logs, got %d", len(logs))
	}
	if logs[1].Level != LogLevelWarn {
		t.Fatalf("expected warn log level, got %s", logs[1].Level)
	}
}
