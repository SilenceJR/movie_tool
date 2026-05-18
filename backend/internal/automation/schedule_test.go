package automation

import (
	"testing"
	"time"
)

func TestNextRunInterval(t *testing.T) {
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	next, err := NextRun(now, ScheduleInterval, "2h")
	if err != nil {
		t.Fatal(err)
	}
	if !next.Equal(now.Add(2 * time.Hour)) {
		t.Fatalf("expected %s, got %s", now.Add(2*time.Hour), next)
	}
}

func TestNextRunDailyCronSameDay(t *testing.T) {
	now := time.Date(2026, 5, 19, 1, 0, 0, 0, time.UTC)
	next, err := NextRun(now, ScheduleCron, "0 3 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if next.Day() != 19 || next.Hour() != 3 || next.Minute() != 0 {
		t.Fatalf("expected same-day 03:00, got %s", next)
	}
}

func TestNextRunDailyCronTomorrow(t *testing.T) {
	now := time.Date(2026, 5, 19, 4, 0, 0, 0, time.UTC)
	next, err := NextRun(now, ScheduleCron, "0 3 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if next.Day() != 20 || next.Hour() != 3 || next.Minute() != 0 {
		t.Fatalf("expected tomorrow 03:00, got %s", next)
	}
}
