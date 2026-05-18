package automation

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreCreateDefaultsEnabled(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	created, err := store.Create(context.Background(), CreateInput{
		Name:         "Scan",
		Type:         TypeScanLibrary,
		ScheduleType: ScheduleInterval,
		Schedule:     "2h",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Enabled {
		t.Fatal("expected automation to be enabled by default")
	}
	if created.NextRunAt == nil || !created.NextRunAt.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("expected next run at %s, got %v", now.Add(2*time.Hour), created.NextRunAt)
	}

	automations, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(automations) != 1 {
		t.Fatalf("expected 1 automation, got %d", len(automations))
	}
}

func TestMemoryStoreUpdatePauseResume(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	created, err := store.Create(context.Background(), CreateInput{
		Name:         "Scan",
		Type:         TypeScanLibrary,
		ScheduleType: ScheduleInterval,
		Schedule:     "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	paused := false
	updated, ok, err := store.Update(context.Background(), created.ID, UpdateInput{Enabled: &paused})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected update to find automation")
	}
	if updated.Enabled || updated.NextRunAt != nil {
		t.Fatalf("expected paused automation without next run, got enabled=%v next=%v", updated.Enabled, updated.NextRunAt)
	}

	now = time.Date(2026, 5, 19, 11, 0, 0, 0, time.UTC)
	resumed := true
	schedule := "30m"
	updated, ok, err = store.Update(context.Background(), created.ID, UpdateInput{
		Enabled:  &resumed,
		Schedule: &schedule,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected update to find automation")
	}
	if !updated.Enabled {
		t.Fatal("expected resumed automation to be enabled")
	}
	if updated.NextRunAt == nil || !updated.NextRunAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("expected next run at %s, got %v", now.Add(30*time.Minute), updated.NextRunAt)
	}
}

func TestMemoryStoreRecordRunHistory(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	created, err := store.Create(context.Background(), CreateInput{
		Name:         "Scan",
		Type:         TypeScanLibrary,
		ScheduleType: ScheduleInterval,
		Schedule:     "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	started := now.Add(5 * time.Minute)
	finished := now.Add(10 * time.Minute)
	now = finished
	run, err := store.RecordRun(context.Background(), RecordRunInput{
		AutomationID: created.ID,
		TaskID:       "task-1",
		Status:       RunSucceeded,
		StartedAt:    &started,
		FinishedAt:   &finished,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunSucceeded || run.TaskID != "task-1" {
		t.Fatalf("unexpected run: %+v", run)
	}

	runs, err := store.ListRuns(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].StartedAt == nil || !runs[0].StartedAt.Equal(started) {
		t.Fatalf("expected run started at %s, got %v", started, runs[0].StartedAt)
	}

	automation, ok, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected automation to exist")
	}
	if automation.LastRunAt == nil || !automation.LastRunAt.Equal(started) {
		t.Fatalf("expected last run at %s, got %v", started, automation.LastRunAt)
	}
	if automation.NextRunAt == nil || !automation.NextRunAt.Equal(finished.Add(time.Hour)) {
		t.Fatalf("expected next run at %s, got %v", finished.Add(time.Hour), automation.NextRunAt)
	}
}
