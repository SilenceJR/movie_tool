package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestQueueRunSuccess(t *testing.T) {
	queue := NewQueue()
	queue.now = func() time.Time {
		return time.Unix(1, 0)
	}
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
}

func TestQueueMissingHandler(t *testing.T) {
	queue := NewQueue()
	created := queue.Enqueue(TypeLibraryScan, "scan")
	done := queue.Run(context.Background(), created.ID)

	if done.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", done.Status)
	}
}
