package organizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLStore struct {
	db SQLDB
}

func NewSQLStore(db SQLDB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) SavePlan(ctx context.Context, plan Plan) (Plan, error) {
	summaryJSON, err := json.Marshal(plan.Summary)
	if err != nil {
		return Plan{}, err
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO organizer_plans (id, library_id, status, dry_run, summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		plan.ID,
		plan.LibraryID,
		string(plan.Status),
		boolInt(plan.DryRun),
		string(summaryJSON),
		formatTime(plan.CreatedAt),
		formatTime(plan.UpdatedAt),
	)
	if err != nil {
		return Plan{}, err
	}

	for _, action := range plan.Actions {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO organizer_actions (
  id, plan_id, media_id, media_file_id, action_type, source_path, target_path, status,
  conflict_reason, error, executed_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			action.ID,
			plan.ID,
			action.MediaID,
			action.MediaFileID,
			string(action.ActionType),
			action.SourcePath,
			action.TargetPath,
			string(action.Status),
			action.ConflictReason,
			action.Error,
			formatNullableTime(action.ExecutedAt),
			formatTime(action.CreatedAt),
		); err != nil {
			return Plan{}, err
		}
	}

	return plan, nil
}

func (s *SQLStore) GetPlan(ctx context.Context, id string) (Plan, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, library_id, status, dry_run, summary, created_at, updated_at
FROM organizer_plans
WHERE id = ?`, id)
	plan, err := scanPlan(row)
	if err == sql.ErrNoRows {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, err
	}
	actions, err := s.ListActions(ctx, id)
	if err != nil {
		return Plan{}, false, err
	}
	plan.Actions = actions
	return plan, true, nil
}

func (s *SQLStore) ListActions(ctx context.Context, planID string) ([]Action, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, plan_id, media_id, media_file_id, action_type, source_path, target_path, status,
       conflict_reason, error, executed_at, created_at
FROM organizer_actions
WHERE plan_id = ?
ORDER BY created_at ASC, id ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		action, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlan(scanner rowScanner) (Plan, error) {
	var plan Plan
	var status string
	var dryRun int
	var summaryJSON sql.NullString
	var createdAt string
	var updatedAt string
	err := scanner.Scan(&plan.ID, &plan.LibraryID, &status, &dryRun, &summaryJSON, &createdAt, &updatedAt)
	if err != nil {
		return Plan{}, err
	}
	plan.Status = PlanStatus(status)
	plan.DryRun = dryRun == 1
	if summaryJSON.Valid && summaryJSON.String != "" {
		if err := json.Unmarshal([]byte(summaryJSON.String), &plan.Summary); err != nil {
			return Plan{}, err
		}
	}
	plan.CreatedAt = parseTime(createdAt)
	plan.UpdatedAt = parseTime(updatedAt)
	return plan, nil
}

func scanAction(scanner rowScanner) (Action, error) {
	var action Action
	var actionType string
	var status string
	var conflictReason sql.NullString
	var actionError sql.NullString
	var executedAt sql.NullString
	var createdAt string
	err := scanner.Scan(
		&action.ID,
		&action.PlanID,
		&action.MediaID,
		&action.MediaFileID,
		&actionType,
		&action.SourcePath,
		&action.TargetPath,
		&status,
		&conflictReason,
		&actionError,
		&executedAt,
		&createdAt,
	)
	if err != nil {
		return Action{}, err
	}
	action.ActionType = ActionMode(actionType)
	action.Status = ActionStatus(status)
	action.ConflictReason = conflictReason.String
	action.Error = actionError.String
	action.ExecutedAt = parseNullableTime(executedAt)
	action.CreatedAt = parseTime(createdAt)
	return action, nil
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
