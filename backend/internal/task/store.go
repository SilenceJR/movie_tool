package task

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store interface {
	Create(context.Context, CreateInput) (Task, error)
	Get(context.Context, string) (Task, bool, error)
	List(context.Context) ([]Task, error)
	ListByQuery(context.Context, Query) ([]Task, error)
	AddLog(context.Context, LogInput) (LogEntry, error)
	ListLogs(context.Context, string) ([]LogEntry, error)
	Update(context.Context, string, UpdateInput) (Task, bool, error)
}

type Query struct {
	Status Status
	Type   Type
}

type CreateInput struct {
	Type     Type
	Status   Status
	Progress int
	Message  string
	Error    string
}

type UpdateInput struct {
	Status   *Status
	Progress *int
	Message  *string
	Error    *string
}

type LogInput struct {
	TaskID  string
	Level   LogLevel
	Message string
}

type MemoryStore struct {
	mu    sync.Mutex
	tasks map[string]Task
	logs  map[string][]LogEntry
	now   func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]Task),
		logs:  make(map[string][]LogEntry),
		now:   time.Now,
	}
}

func (s *MemoryStore) Create(_ context.Context, input CreateInput) (Task, error) {
	if input.Type == "" {
		return Task{}, fmt.Errorf("task type is required")
	}
	if input.Status == "" {
		input.Status = StatusPending
	}
	now := s.now().UTC()
	task := Task{
		ID:        fmt.Sprintf("task_%d", now.UnixNano()),
		Type:      input.Type,
		Status:    input.Status,
		Progress:  input.Progress,
		Message:   input.Message,
		Error:     input.Error,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return task, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	return task, ok, nil
}

func (s *MemoryStore) List(context.Context) ([]Task, error) {
	return s.ListByQuery(context.Background(), Query{})
}

func (s *MemoryStore) ListByQuery(_ context.Context, query Query) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		if query.Status != "" && task.Status != query.Status {
			continue
		}
		if query.Type != "" && task.Type != query.Type {
			continue
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}

func (s *MemoryStore) AddLog(_ context.Context, input LogInput) (LogEntry, error) {
	if input.TaskID == "" {
		return LogEntry{}, fmt.Errorf("task id is required")
	}
	if input.Message == "" {
		return LogEntry{}, fmt.Errorf("log message is required")
	}
	if input.Level == "" {
		input.Level = LogLevelInfo
	}
	now := s.now().UTC()
	entry := LogEntry{
		ID:        fmt.Sprintf("task_log_%d", now.UnixNano()),
		TaskID:    input.TaskID,
		Level:     input.Level,
		Message:   input.Message,
		CreatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[input.TaskID]; !ok {
		return LogEntry{}, fmt.Errorf("task not found")
	}
	s.logs[input.TaskID] = append(s.logs[input.TaskID], entry)
	return entry, nil
}

func (s *MemoryStore) ListLogs(_ context.Context, taskID string) ([]LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := append([]LogEntry(nil), s.logs[taskID]...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input UpdateInput) (Task, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return Task{}, false, nil
	}
	if input.Status != nil {
		task.Status = *input.Status
	}
	if input.Progress != nil {
		task.Progress = *input.Progress
	}
	if input.Message != nil {
		task.Message = *input.Message
	}
	if input.Error != nil {
		task.Error = *input.Error
	}
	task.UpdatedAt = s.now().UTC()
	s.tasks[id] = task
	return task, true, nil
}
