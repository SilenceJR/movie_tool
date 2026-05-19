package task

import (
	"context"
)

type Handler func(context.Context, Task) error

type Queue struct {
	store    Store
	handlers map[Type]Handler
}

func NewQueue() *Queue {
	return NewQueueWithStore(NewMemoryStore())
}

func NewQueueWithStore(store Store) *Queue {
	return &Queue{
		store:    store,
		handlers: make(map[Type]Handler),
	}
}

func (q *Queue) Register(taskType Type, handler Handler) {
	q.handlers[taskType] = handler
}

func (q *Queue) Enqueue(taskType Type, message string) Task {
	task, err := q.store.Create(context.Background(), CreateInput{
		Type:    taskType,
		Status:  StatusPending,
		Message: message,
	})
	if err != nil {
		return Task{Type: taskType, Status: StatusFailed, Message: message, Error: err.Error()}
	}
	q.addLog(context.Background(), task.ID, LogLevelInfo, "task queued")
	return task
}

func (q *Queue) Run(ctx context.Context, id string) Task {
	task, ok, err := q.store.Get(ctx, id)
	if err != nil {
		return Task{ID: id, Status: StatusFailed, Error: err.Error()}
	}
	if !ok {
		return Task{ID: id, Status: StatusFailed, Error: "task not found"}
	}
	handler, ok := q.handlers[task.Type]
	if !ok {
		status := StatusFailed
		taskError := "task handler not registered"
		updated, _, _ := q.store.Update(ctx, id, UpdateInput{Status: &status, Error: &taskError})
		q.addLog(ctx, id, LogLevelError, taskError)
		return updated
	}
	running := StatusRunning
	task, _, _ = q.store.Update(ctx, id, UpdateInput{Status: &running})
	q.addLog(ctx, id, LogLevelInfo, "task started")

	err = handler(ctx, task)
	if err != nil {
		failed := StatusFailed
		taskError := err.Error()
		updated, _, _ := q.store.Update(ctx, id, UpdateInput{Status: &failed, Error: &taskError})
		q.addLog(ctx, id, LogLevelError, taskError)
		return updated
	}
	succeeded := StatusSucceeded
	progress := 100
	updated, _, _ := q.store.Update(ctx, id, UpdateInput{Status: &succeeded, Progress: &progress})
	q.addLog(ctx, id, LogLevelInfo, "task succeeded")
	return updated
}

func (q *Queue) Get(id string) (Task, bool) {
	task, ok, _ := q.store.Get(context.Background(), id)
	return task, ok
}

func (q *Queue) Cancel(id string) (Task, bool) {
	status := StatusCanceled
	task, ok, _ := q.store.Update(context.Background(), id, UpdateInput{Status: &status})
	if ok {
		q.addLog(context.Background(), id, LogLevelWarn, "task canceled")
	}
	return task, ok
}

func (q *Queue) List() []Task {
	tasks, _ := q.store.List(context.Background())
	return tasks
}

func (q *Queue) ListByQuery(query Query) []Task {
	tasks, _ := q.store.ListByQuery(context.Background(), query)
	return tasks
}

func (q *Queue) Logs(id string) []LogEntry {
	entries, _ := q.store.ListLogs(context.Background(), id)
	return entries
}

func (q *Queue) Log(id string, level LogLevel, message string) {
	q.addLog(context.Background(), id, level, message)
}

func (q *Queue) addLog(ctx context.Context, id string, level LogLevel, message string) {
	_, _ = q.store.AddLog(ctx, LogInput{
		TaskID:  id,
		Level:   level,
		Message: message,
	})
}
