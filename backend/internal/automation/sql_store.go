package automation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

func (s *SQLStore) List(ctx context.Context) ([]Automation, error) {
	return s.ListByQuery(ctx, Query{})
}

func (s *SQLStore) ListByQuery(ctx context.Context, query Query) ([]Automation, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 2)
	if query.Type != "" {
		where += " AND automation_type = ?"
		args = append(args, string(query.Type))
	}
	if query.Enabled != nil {
		where += " AND enabled = ?"
		args = append(args, boolInt(*query.Enabled))
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, automation_type, schedule_type, schedule, scope, options, enabled, last_run_at, next_run_at, created_at, updated_at
FROM automations
`+where+`
ORDER BY created_at ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var automations []Automation
	for rows.Next() {
		automation, err := scanAutomation(rows)
		if err != nil {
			return nil, err
		}
		automations = append(automations, automation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return automations, nil
}

func (s *SQLStore) Get(ctx context.Context, id string) (Automation, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, automation_type, schedule_type, schedule, scope, options, enabled, last_run_at, next_run_at, created_at, updated_at
FROM automations
WHERE id = ?`, id)
	automation, err := scanAutomation(row)
	if err == sql.ErrNoRows {
		return Automation{}, false, nil
	}
	if err != nil {
		return Automation{}, false, err
	}
	return automation, true, nil
}

func (s *SQLStore) Create(ctx context.Context, input CreateInput) (Automation, error) {
	if err := validateCreateInput(input); err != nil {
		return Automation{}, err
	}

	now := s.now().UTC()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	nextRunAt, err := nextRunAt(now, enabled, input.ScheduleType, input.Schedule)
	if err != nil {
		return Automation{}, err
	}

	automation := Automation{
		ID:           newID("auto"),
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

	scopeJSON, err := marshalJSON(automation.Scope)
	if err != nil {
		return Automation{}, err
	}
	optionsJSON, err := marshalJSON(automation.Options)
	if err != nil {
		return Automation{}, err
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO automations (
  id, name, automation_type, schedule_type, schedule, scope, options, enabled, last_run_at, next_run_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		automation.ID,
		automation.Name,
		string(automation.Type),
		string(automation.ScheduleType),
		automation.Schedule,
		scopeJSON,
		optionsJSON,
		boolInt(automation.Enabled),
		formatNullableTime(automation.LastRunAt),
		formatNullableTime(automation.NextRunAt),
		formatTime(automation.CreatedAt),
		formatTime(automation.UpdatedAt),
	)
	if err != nil {
		return Automation{}, err
	}
	return automation, nil
}

func (s *SQLStore) Update(ctx context.Context, id string, input UpdateInput) (Automation, bool, error) {
	automation, ok, err := s.Get(ctx, id)
	if err != nil || !ok {
		return Automation{}, ok, err
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

	now := s.now().UTC()
	if recalculateNextRun {
		nextRunAt, err := nextRunAt(now, automation.Enabled, automation.ScheduleType, automation.Schedule)
		if err != nil {
			return Automation{}, true, err
		}
		automation.NextRunAt = nextRunAt
	}
	automation.UpdatedAt = now

	scopeJSON, err := marshalJSON(automation.Scope)
	if err != nil {
		return Automation{}, true, err
	}
	optionsJSON, err := marshalJSON(automation.Options)
	if err != nil {
		return Automation{}, true, err
	}

	_, err = s.db.ExecContext(ctx, `
UPDATE automations
SET name = ?, automation_type = ?, schedule_type = ?, schedule = ?, scope = ?, options = ?,
    enabled = ?, last_run_at = ?, next_run_at = ?, updated_at = ?
WHERE id = ?`,
		automation.Name,
		string(automation.Type),
		string(automation.ScheduleType),
		automation.Schedule,
		scopeJSON,
		optionsJSON,
		boolInt(automation.Enabled),
		formatNullableTime(automation.LastRunAt),
		formatNullableTime(automation.NextRunAt),
		formatTime(automation.UpdatedAt),
		automation.ID,
	)
	if err != nil {
		return Automation{}, true, err
	}
	return automation, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM automation_runs WHERE automation_id = ?`, id); err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM automations WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *SQLStore) RecordRun(ctx context.Context, input RecordRunInput) (Run, error) {
	if input.AutomationID == "" {
		return Run{}, fmt.Errorf("automation id is required")
	}
	if input.Status == "" {
		input.Status = RunPending
	}

	automation, ok, err := s.Get(ctx, input.AutomationID)
	if err != nil {
		return Run{}, err
	}
	if !ok {
		return Run{}, fmt.Errorf("automation %q not found", input.AutomationID)
	}

	now := s.now().UTC()
	run := Run{
		ID:           newID("autorun"),
		AutomationID: input.AutomationID,
		TaskID:       input.TaskID,
		Status:       input.Status,
		StartedAt:    cloneTime(input.StartedAt),
		FinishedAt:   cloneTime(input.FinishedAt),
		Error:        input.Error,
		CreatedAt:    now,
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO automation_runs (id, automation_id, task_id, status, started_at, finished_at, error, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		run.AutomationID,
		run.TaskID,
		string(run.Status),
		formatNullableTime(run.StartedAt),
		formatNullableTime(run.FinishedAt),
		run.Error,
		formatTime(run.CreatedAt),
	)
	if err != nil {
		return Run{}, err
	}

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
	if _, _, err := s.Update(ctx, automation.ID, UpdateInput{
		Name:         &automation.Name,
		Type:         &automation.Type,
		ScheduleType: &automation.ScheduleType,
		Schedule:     &automation.Schedule,
		Scope:        &automation.Scope,
		Options:      &automation.Options,
		Enabled:      &automation.Enabled,
	}); err != nil {
		return Run{}, err
	}
	// Update preserves LastRunAt from the loaded automation only if we write it directly.
	_, err = s.db.ExecContext(ctx, `UPDATE automations SET last_run_at = ?, next_run_at = ?, updated_at = ? WHERE id = ?`,
		formatNullableTime(automation.LastRunAt),
		formatNullableTime(automation.NextRunAt),
		formatTime(automation.UpdatedAt),
		automation.ID,
	)
	if err != nil {
		return Run{}, err
	}

	return run, nil
}

func (s *SQLStore) ListRuns(ctx context.Context, automationID string) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, automation_id, task_id, status, started_at, finished_at, error, created_at
FROM automation_runs
WHERE automation_id = ?
ORDER BY created_at ASC, id ASC`, automationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

type automationScanner interface {
	Scan(dest ...any) error
}

func scanAutomation(scanner automationScanner) (Automation, error) {
	var automation Automation
	var automationType string
	var scheduleType string
	var scopeJSON sql.NullString
	var optionsJSON sql.NullString
	var enabled int
	var lastRunAt sql.NullString
	var nextRunAt sql.NullString
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&automation.ID,
		&automation.Name,
		&automationType,
		&scheduleType,
		&automation.Schedule,
		&scopeJSON,
		&optionsJSON,
		&enabled,
		&lastRunAt,
		&nextRunAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Automation{}, err
	}
	automation.Type = Type(automationType)
	automation.ScheduleType = ScheduleType(scheduleType)
	if scopeJSON.Valid && scopeJSON.String != "" {
		if err := json.Unmarshal([]byte(scopeJSON.String), &automation.Scope); err != nil {
			return Automation{}, err
		}
	}
	if optionsJSON.Valid && optionsJSON.String != "" {
		if err := json.Unmarshal([]byte(optionsJSON.String), &automation.Options); err != nil {
			return Automation{}, err
		}
	}
	automation.Enabled = enabled == 1
	automation.LastRunAt = parseNullableTime(lastRunAt)
	automation.NextRunAt = parseNullableTime(nextRunAt)
	automation.CreatedAt = parseTime(createdAt)
	automation.UpdatedAt = parseTime(updatedAt)
	return automation, nil
}

func scanRun(scanner automationScanner) (Run, error) {
	var run Run
	var status string
	var startedAt sql.NullString
	var finishedAt sql.NullString
	var taskID sql.NullString
	var runError sql.NullString
	var createdAt string
	err := scanner.Scan(&run.ID, &run.AutomationID, &taskID, &status, &startedAt, &finishedAt, &runError, &createdAt)
	if err != nil {
		return Run{}, err
	}
	run.TaskID = taskID.String
	run.Status = RunStatus(status)
	run.StartedAt = parseNullableTime(startedAt)
	run.FinishedAt = parseNullableTime(finishedAt)
	run.Error = runError.String
	run.CreatedAt = parseTime(createdAt)
	return run, nil
}

func validateCreateInput(input CreateInput) error {
	if input.Name == "" {
		return fmt.Errorf("automation name is required")
	}
	if input.Type == "" {
		return fmt.Errorf("automation type is required")
	}
	if input.ScheduleType == "" {
		return fmt.Errorf("automation schedule type is required")
	}
	if input.Schedule == "" {
		return fmt.Errorf("automation schedule is required")
	}
	return nil
}

func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatNullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return formatTime(*value)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseNullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed := parseTime(value.String)
	return &parsed
}
