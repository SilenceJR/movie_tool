package task

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Handler func(context.Context, Task) error

type Queue struct {
	mu       sync.Mutex
	tasks    map[string]Task
	handlers map[Type]Handler
	now      func() time.Time
}

func NewQueue() *Queue {
	return &Queue{
		tasks:    make(map[string]Task),
		handlers: make(map[Type]Handler),
		now:      time.Now,
	}
}

func (q *Queue) Register(taskType Type, handler Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = handler
}

func (q *Queue) Enqueue(taskType Type, message string) Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.now()
	task := Task{
		ID:        fmt.Sprintf("%d", now.UnixNano()),
		Type:      taskType,
		Status:    StatusPending,
		Progress:  0,
		Message:   message,
		CreatedAt: now,
		UpdatedAt: now,
	}
	q.tasks[task.ID] = task
	return task
}

func (q *Queue) Run(ctx context.Context, id string) Task {
	q.mu.Lock()
	task, ok := q.tasks[id]
	if !ok {
		q.mu.Unlock()
		return Task{ID: id, Status: StatusFailed, Error: "task not found"}
	}
	handler, ok := q.handlers[task.Type]
	if !ok {
		task.Status = StatusFailed
		task.Error = "task handler not registered"
		task.UpdatedAt = q.now()
		q.tasks[id] = task
		q.mu.Unlock()
		return task
	}
	task.Status = StatusRunning
	task.UpdatedAt = q.now()
	q.tasks[id] = task
	q.mu.Unlock()

	err := handler(ctx, task)

	q.mu.Lock()
	defer q.mu.Unlock()
	task = q.tasks[id]
	task.UpdatedAt = q.now()
	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		q.tasks[id] = task
		return task
	}
	task.Status = StatusSucceeded
	task.Progress = 100
	q.tasks[id] = task
	return task
}

func (q *Queue) Get(id string) (Task, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.tasks[id]
	return task, ok
}

func (q *Queue) List() []Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	tasks := make([]Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}
