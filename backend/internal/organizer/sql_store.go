package organizer

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
	db SQLDB
}

func NewSQLStore(db SQLDB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) CreateRule(ctx context.Context, input RuleInput) (Rule, error) {
	now := time.Now().UTC()
	rule, err := ruleFromInput(input, now)
	if err != nil {
		return Rule{}, err
	}
	rule.ID = newID("organizer_rule")
	_, err = s.db.ExecContext(ctx, `
INSERT INTO organizer_rules (
  id, name, library_id, media_type, target_root, folder_template, file_template,
  sidecar_policy, action_mode, conflict_policy, enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID,
		rule.Name,
		nullableString(rule.LibraryID),
		nullableString(rule.MediaType),
		rule.TargetRoot,
		rule.FolderTemplate,
		rule.FileTemplate,
		rule.SidecarPolicy,
		string(rule.ActionMode),
		string(rule.ConflictPolicy),
		boolInt(rule.Enabled),
		formatTime(rule.CreatedAt),
		formatTime(rule.UpdatedAt),
	)
	if err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func (s *SQLStore) GetRule(ctx context.Context, id string) (Rule, bool, error) {
	rule, err := scanRule(s.db.QueryRowContext(ctx, `
SELECT id, name, library_id, media_type, target_root, folder_template, file_template,
       sidecar_policy, action_mode, conflict_policy, enabled, created_at, updated_at
FROM organizer_rules
WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Rule{}, false, nil
	}
	if err != nil {
		return Rule{}, false, err
	}
	return rule, true, nil
}

func (s *SQLStore) ListRules(ctx context.Context) ([]Rule, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, library_id, media_type, target_root, folder_template, file_template,
       sidecar_policy, action_mode, conflict_policy, enabled, created_at, updated_at
FROM organizer_rules
ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *SQLStore) UpdateRule(ctx context.Context, id string, input RuleUpdate) (Rule, bool, error) {
	rule, ok, err := s.GetRule(ctx, id)
	if err != nil || !ok {
		return Rule{}, ok, err
	}
	applyRuleUpdate(&rule, input)
	if err := validateRule(rule); err != nil {
		return Rule{}, true, err
	}
	rule.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE organizer_rules
SET name = ?, library_id = ?, media_type = ?, target_root = ?, folder_template = ?,
    file_template = ?, sidecar_policy = ?, action_mode = ?, conflict_policy = ?,
    enabled = ?, updated_at = ?
WHERE id = ?`,
		rule.Name,
		nullableString(rule.LibraryID),
		nullableString(rule.MediaType),
		rule.TargetRoot,
		rule.FolderTemplate,
		rule.FileTemplate,
		rule.SidecarPolicy,
		string(rule.ActionMode),
		string(rule.ConflictPolicy),
		boolInt(rule.Enabled),
		formatTime(rule.UpdatedAt),
		rule.ID,
	)
	if err != nil {
		return Rule{}, true, err
	}
	return rule, true, nil
}

func (s *SQLStore) DeleteRule(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM organizer_rules WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
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

func (s *SQLStore) UpdatePlan(ctx context.Context, plan Plan) (Plan, error) {
	if plan.ID == "" {
		return Plan{}, fmt.Errorf("plan id is required")
	}
	summaryJSON, err := json.Marshal(plan.Summary)
	if err != nil {
		return Plan{}, err
	}

	result, err := s.db.ExecContext(ctx, `
UPDATE organizer_plans
SET status = ?, dry_run = ?, summary = ?, updated_at = ?
WHERE id = ?`,
		string(plan.Status),
		boolInt(plan.DryRun),
		string(summaryJSON),
		formatTime(plan.UpdatedAt),
		plan.ID,
	)
	if err != nil {
		return Plan{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Plan{}, err
	}
	if affected == 0 {
		return Plan{}, fmt.Errorf("organizer plan not found")
	}

	for _, action := range plan.Actions {
		if _, err := s.db.ExecContext(ctx, `
UPDATE organizer_actions
SET target_path = ?, status = ?, conflict_reason = ?, error = ?, executed_at = ?
WHERE id = ? AND plan_id = ?`,
			action.TargetPath,
			string(action.Status),
			nullableString(action.ConflictReason),
			nullableString(action.Error),
			formatNullableTime(action.ExecutedAt),
			action.ID,
			plan.ID,
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

func scanRule(scanner rowScanner) (Rule, error) {
	var rule Rule
	var libraryID sql.NullString
	var mediaType sql.NullString
	var actionMode string
	var conflictPolicy string
	var enabled int
	var createdAt string
	var updatedAt string
	err := scanner.Scan(
		&rule.ID,
		&rule.Name,
		&libraryID,
		&mediaType,
		&rule.TargetRoot,
		&rule.FolderTemplate,
		&rule.FileTemplate,
		&rule.SidecarPolicy,
		&actionMode,
		&conflictPolicy,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Rule{}, err
	}
	rule.LibraryID = libraryID.String
	rule.MediaType = mediaType.String
	rule.ActionMode = ActionMode(actionMode)
	rule.ConflictPolicy = ConflictPolicy(conflictPolicy)
	rule.Enabled = enabled == 1
	rule.CreatedAt = parseTime(createdAt)
	rule.UpdatedAt = parseTime(updatedAt)
	return rule, nil
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
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
