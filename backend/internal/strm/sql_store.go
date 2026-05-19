package strm

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

func (s *SQLStore) CreateRule(ctx context.Context, input RuleInput) (Rule, error) {
	if err := validateRule(input.Name, input.SourcePrefix, input.TargetPrefix, input.OutputPath); err != nil {
		return Rule{}, err
	}
	now := s.now().UTC()
	rule := Rule{
		ID:           newID("strm_rule"),
		Name:         input.Name,
		SourcePrefix: input.SourcePrefix,
		TargetPrefix: input.TargetPrefix,
		OutputPath:   input.OutputPath,
		Enabled:      input.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO strm_rules (id, name, source_prefix, target_prefix, output_path, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath, boolInt(rule.Enabled), formatTime(rule.CreatedAt), formatTime(rule.UpdatedAt))
	if err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func (s *SQLStore) GetRule(ctx context.Context, id string) (Rule, bool, error) {
	rule, err := scanRule(s.db.QueryRowContext(ctx, `
SELECT id, name, source_prefix, target_prefix, output_path, enabled, created_at, updated_at
FROM strm_rules WHERE id = ?`, id))
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
SELECT id, name, source_prefix, target_prefix, output_path, enabled, created_at, updated_at
FROM strm_rules
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
	return rules, rows.Err()
}

func (s *SQLStore) UpdateRule(ctx context.Context, id string, input RuleUpdate) (Rule, bool, error) {
	rule, ok, err := s.GetRule(ctx, id)
	if err != nil || !ok {
		return Rule{}, ok, err
	}
	applyRuleUpdate(&rule, input)
	if err := validateRule(rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath); err != nil {
		return Rule{}, true, err
	}
	rule.UpdatedAt = s.now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE strm_rules
SET name = ?, source_prefix = ?, target_prefix = ?, output_path = ?, enabled = ?, updated_at = ?
WHERE id = ?`,
		rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath, boolInt(rule.Enabled), formatTime(rule.UpdatedAt), rule.ID)
	if err != nil {
		return Rule{}, true, err
	}
	return rule, true, nil
}

func (s *SQLStore) DeleteRule(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM strm_rules WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRule(scanner scanner) (Rule, error) {
	var rule Rule
	var enabled int
	var createdAt, updatedAt string
	if err := scanner.Scan(&rule.ID, &rule.Name, &rule.SourcePrefix, &rule.TargetPrefix, &rule.OutputPath, &enabled, &createdAt, &updatedAt); err != nil {
		return Rule{}, err
	}
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

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
