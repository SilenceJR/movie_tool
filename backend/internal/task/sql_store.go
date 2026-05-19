package task

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLStore struct {
	db  SQLDB
	now func() time.Time
}

func NewSQLStore(db SQLDB) *SQLStore {
	return &SQLStore{db: db, now: time.Now}
}

func (s *SQLStore) Create(ctx context.Context, input CreateInput) (Task, error) {
	if input.Type == "" {
		return Task{}, fmt.Errorf("task type is required")
	}
	if input.Status == "" {
		input.Status = StatusPending
	}
	now := s.now().UTC()
	task := Task{
		ID:        newID("task"),
		Type:      input.Type,
		Status:    input.Status,
		Progress:  input.Progress,
		Message:   input.Message,
		Error:     input.Error,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tasks (id, task_type, status, progress, message, error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, string(task.Type), string(task.Status), task.Progress, task.Message, task.Error, formatTime(task.CreatedAt), formatTime(task.UpdatedAt),
	)
	if err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *SQLStore) Get(ctx context.Context, id string) (Task, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, task_type, status, progress, message, error, created_at, updated_at
FROM tasks WHERE id = ?`, id)
	task, err := scanTask(row)
	if err == sql.ErrNoRows {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}
	return task, true, nil
}

func (s *SQLStore) List(ctx context.Context) ([]Task, error) {
	return s.ListByQuery(ctx, Query{})
}

func (s *SQLStore) ListByQuery(ctx context.Context, query Query) ([]Task, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 2)
	if query.Status != "" {
		where += " AND status = ?"
		args = append(args, string(query.Status))
	}
	if query.Type != "" {
		where += " AND task_type = ?"
		args = append(args, string(query.Type))
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, task_type, status, progress, message, error, created_at, updated_at
FROM tasks
`+where+`
ORDER BY created_at ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *SQLStore) AddLog(ctx context.Context, input LogInput) (LogEntry, error) {
	if input.TaskID == "" {
		return LogEntry{}, fmt.Errorf("task id is required")
	}
	if input.Message == "" {
		return LogEntry{}, fmt.Errorf("log message is required")
	}
	if input.Level == "" {
		input.Level = LogLevelInfo
	}
	entry := LogEntry{
		ID:        newID("task_log"),
		TaskID:    input.TaskID,
		Level:     input.Level,
		Message:   input.Message,
		CreatedAt: s.now().UTC(),
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO task_logs (id, task_id, level, message, created_at)
VALUES (?, ?, ?, ?, ?)`,
		entry.ID, entry.TaskID, string(entry.Level), entry.Message, formatTime(entry.CreatedAt),
	)
	if err != nil {
		return LogEntry{}, err
	}
	return entry, nil
}

func (s *SQLStore) ListLogs(ctx context.Context, taskID string) ([]LogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, task_id, level, message, created_at
FROM task_logs
WHERE task_id = ?
ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []LogEntry
	for rows.Next() {
		entry, err := scanLogEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *SQLStore) Update(ctx context.Context, id string, input UpdateInput) (Task, bool, error) {
	task, ok, err := s.Get(ctx, id)
	if err != nil || !ok {
		return Task{}, ok, err
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
	_, err = s.db.ExecContext(ctx, `
UPDATE tasks SET status = ?, progress = ?, message = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(task.Status), task.Progress, task.Message, task.Error, formatTime(task.UpdatedAt), task.ID,
	)
	if err != nil {
		return Task{}, true, err
	}
	return task, true, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (Task, error) {
	var task Task
	var taskType string
	var status string
	var message sql.NullString
	var taskError sql.NullString
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(&task.ID, &taskType, &status, &task.Progress, &message, &taskError, &createdAt, &updatedAt); err != nil {
		return Task{}, err
	}
	task.Type = Type(taskType)
	task.Status = Status(status)
	task.Message = message.String
	task.Error = taskError.String
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(updatedAt)
	return task, nil
}

func scanLogEntry(scanner taskScanner) (LogEntry, error) {
	var entry LogEntry
	var level string
	var createdAt string
	if err := scanner.Scan(&entry.ID, &entry.TaskID, &level, &entry.Message, &createdAt); err != nil {
		return LogEntry{}, err
	}
	entry.Level = LogLevel(level)
	entry.CreatedAt = parseTime(createdAt)
	return entry, nil
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
