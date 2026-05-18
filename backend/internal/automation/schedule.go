package automation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NextRun(now time.Time, scheduleType ScheduleType, schedule string) (time.Time, error) {
	switch scheduleType {
	case ScheduleInterval:
		duration, err := time.ParseDuration(schedule)
		if err != nil {
			return time.Time{}, err
		}
		if duration <= 0 {
			return time.Time{}, fmt.Errorf("interval must be positive")
		}
		return now.Add(duration), nil
	case ScheduleCron:
		return nextDailyCron(now, schedule)
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule type %q", scheduleType)
	}
}

func nextDailyCron(now time.Time, schedule string) (time.Time, error) {
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron schedule must have 5 fields")
	}

	minute, err := strconv.Atoi(fields[0])
	if err != nil {
		return time.Time{}, err
	}
	hour, err := strconv.Atoi(fields[1])
	if err != nil {
		return time.Time{}, err
	}
	if minute < 0 || minute > 59 || hour < 0 || hour > 23 {
		return time.Time{}, fmt.Errorf("cron hour or minute out of range")
	}
	if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
		return time.Time{}, fmt.Errorf("only daily cron schedules are supported initially")
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
}
