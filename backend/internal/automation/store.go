package automation

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store interface {
	List(context.Context) ([]Automation, error)
	ListByQuery(context.Context, Query) ([]Automation, error)
	Get(context.Context, string) (Automation, bool, error)
	Create(context.Context, CreateInput) (Automation, error)
	Update(context.Context, string, UpdateInput) (Automation, bool, error)
	Delete(context.Context, string) (bool, error)
	RecordRun(context.Context, RecordRunInput) (Run, error)
	ListRuns(context.Context, string) ([]Run, error)
}

type Query struct {
	Type    Type
	Enabled *bool
}

type CreateInput struct {
	Name         string       `json:"name"`
	Type         Type         `json:"automation_type"`
	ScheduleType ScheduleType `json:"schedule_type"`
	Schedule     string       `json:"schedule"`
	Scope        Scope        `json:"scope"`
	Options      Options      `json:"options"`
	Enabled      *bool        `json:"enabled"`
}

type UpdateInput struct {
	Name         *string       `json:"name"`
	Type         *Type         `json:"automation_type"`
	ScheduleType *ScheduleType `json:"schedule_type"`
	Schedule     *string       `json:"schedule"`
	Scope        *Scope        `json:"scope"`
	Options      *Options      `json:"options"`
	Enabled      *bool         `json:"enabled"`
}

type RecordRunInput struct {
	AutomationID string     `json:"automation_id"`
	TaskID       string     `json:"task_id"`
	Status       RunStatus  `json:"status"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Error        string     `json:"error"`
}

type MemoryStore struct {
	mu          sync.Mutex
	automations map[string]Automation
	runs        map[string][]Run
	now         func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		automations: make(map[string]Automation),
		runs:        make(map[string][]Run),
		now:         time.Now,
	}
}

func (s *MemoryStore) List(context.Context) ([]Automation, error) {
	return s.ListByQuery(context.Background(), Query{})
}

func (s *MemoryStore) ListByQuery(_ context.Context, query Query) ([]Automation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	automations := make([]Automation, 0, len(s.automations))
	for _, automation := range s.automations {
		if query.Type != "" && automation.Type != query.Type {
			continue
		}
		if query.Enabled != nil && automation.Enabled != *query.Enabled {
			continue
		}
		automations = append(automations, cloneAutomation(automation))
	}
	sort.Slice(automations, func(i, j int) bool {
		if automations[i].CreatedAt.Equal(automations[j].CreatedAt) {
			return automations[i].ID < automations[j].ID
		}
		return automations[i].CreatedAt.Before(automations[j].CreatedAt)
	})
	return automations, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Automation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	automation, ok := s.automations[id]
	return cloneAutomation(automation), ok, nil
}

func (s *MemoryStore) Create(_ context.Context, input CreateInput) (Automation, error) {
	if input.Name == "" {
		return Automation{}, fmt.Errorf("automation name is required")
	}
	if input.Type == "" {
		return Automation{}, fmt.Errorf("automation type is required")
	}
	if input.ScheduleType == "" {
		return Automation{}, fmt.Errorf("automation schedule type is required")
	}
	if input.Schedule == "" {
		return Automation{}, fmt.Errorf("automation schedule is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	nextRunAt, err := nextRunAt(now, enabled, input.ScheduleType, input.Schedule)
	if err != nil {
		return Automation{}, err
	}

	automation := Automation{
		ID:           fmt.Sprintf("%d", now.UnixNano()),
		Name:         input.Name,
		Type:         input.Type,
		ScheduleType: input.ScheduleType,
		Schedule:     input.Schedule,
		Scope:        cloneScope(input.Scope),
		Options:      input.Options,
		Enabled:      enabled,
		NextRunAt:    nextRunAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.automations[automation.ID] = automation
	return cloneAutomation(automation), nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input UpdateInput) (Automation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	automation, ok := s.automations[id]
	if !ok {
		return Automation{}, false, nil
	}

	recalculateNextRun := false
	if input.Name != nil {
		automation.Name = *input.Name
	}
	if input.Type != nil {
		automation.Type = *input.Type
	}
	if input.ScheduleType != nil {
		automation.ScheduleType = *input.ScheduleType
		recalculateNextRun = true
	}
	if input.Schedule != nil {
		automation.Schedule = *input.Schedule
		recalculateNextRun = true
	}
	if input.Scope != nil {
		automation.Scope = cloneScope(*input.Scope)
	}
	if input.Options != nil {
		automation.Options = *input.Options
	}
	if input.Enabled != nil {
		automation.Enabled = *input.Enabled
		recalculateNextRun = true
	}

	now := s.now()
	if recalculateNextRun {
		nextRunAt, err := nextRunAt(now, automation.Enabled, automation.ScheduleType, automation.Schedule)
		if err != nil {
			return Automation{}, true, err
		}
		automation.NextRunAt = nextRunAt
	}
	automation.UpdatedAt = now
	s.automations[id] = automation
	return cloneAutomation(automation), true, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.automations[id]; !ok {
		return false, nil
	}
	delete(s.automations, id)
	delete(s.runs, id)
	return true, nil
}

func (s *MemoryStore) RecordRun(_ context.Context, input RecordRunInput) (Run, error) {
	if input.AutomationID == "" {
		return Run{}, fmt.Errorf("automation id is required")
	}
	if input.Status == "" {
		input.Status = RunPending
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	automation, ok := s.automations[input.AutomationID]
	if !ok {
		return Run{}, fmt.Errorf("automation %q not found", input.AutomationID)
	}

	now := s.now()
	run := Run{
		ID:           fmt.Sprintf("%d", now.UnixNano()),
		AutomationID: input.AutomationID,
		TaskID:       input.TaskID,
		Status:       input.Status,
		StartedAt:    cloneTime(input.StartedAt),
		FinishedAt:   cloneTime(input.FinishedAt),
		Error:        input.Error,
		CreatedAt:    now,
	}
	s.runs[input.AutomationID] = append(s.runs[input.AutomationID], run)

	if input.StartedAt != nil {
		automation.LastRunAt = cloneTime(input.StartedAt)
	} else {
		automation.LastRunAt = &now
	}
	nextRunAt, err := nextRunAt(now, automation.Enabled, automation.ScheduleType, automation.Schedule)
	if err != nil {
		return Run{}, err
	}
	automation.NextRunAt = nextRunAt
	automation.UpdatedAt = now
	s.automations[input.AutomationID] = automation

	return cloneRun(run), nil
}

func (s *MemoryStore) ListRuns(_ context.Context, automationID string) ([]Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runs := s.runs[automationID]
	result := make([]Run, 0, len(runs))
	for _, run := range runs {
		result = append(result, cloneRun(run))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func nextRunAt(now time.Time, enabled bool, scheduleType ScheduleType, schedule string) (*time.Time, error) {
	if !enabled {
		return nil, nil
	}
	next, err := NextRun(now, scheduleType, schedule)
	if err != nil {
		return nil, err
	}
	return &next, nil
}

func cloneAutomation(automation Automation) Automation {
	automation.Scope = cloneScope(automation.Scope)
	automation.LastRunAt = cloneTime(automation.LastRunAt)
	automation.NextRunAt = cloneTime(automation.NextRunAt)
	return automation
}

func cloneScope(scope Scope) Scope {
	scope.MediaIDs = append([]string(nil), scope.MediaIDs...)
	return scope
}

func cloneRun(run Run) Run {
	run.StartedAt = cloneTime(run.StartedAt)
	run.FinishedAt = cloneTime(run.FinishedAt)
	return run
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
