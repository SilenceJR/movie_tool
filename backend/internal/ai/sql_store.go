package ai

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

func (s *SQLStore) CreateProvider(ctx context.Context, input ProviderInput) (Provider, error) {
	provider, err := providerFromInput(input, s.now().UTC())
	if err != nil {
		return Provider{}, err
	}
	provider.ID = newID("ai_provider")
	provider.HasAPIKey = input.APIKey != ""
	_, err = s.db.ExecContext(ctx, `
INSERT INTO ai_providers (id, name, provider_type, base_url, api_key_encrypted, default_model, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider.ID, provider.Name, string(provider.Type), nullString(provider.BaseURL), nullString(input.APIKey),
		nullString(provider.DefaultModel), boolInt(provider.Enabled), formatTime(provider.CreatedAt), formatTime(provider.UpdatedAt))
	if err != nil {
		return Provider{}, err
	}
	return provider, nil
}

func (s *SQLStore) GetProvider(ctx context.Context, id string) (Provider, bool, error) {
	provider, err := scanProvider(s.db.QueryRowContext(ctx, `
SELECT id, name, provider_type, base_url, api_key_encrypted, default_model, enabled, created_at, updated_at
FROM ai_providers
WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Provider{}, false, nil
	}
	if err != nil {
		return Provider{}, false, err
	}
	return provider, true, nil
}

func (s *SQLStore) ListProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, provider_type, base_url, api_key_encrypted, default_model, enabled, created_at, updated_at
FROM ai_providers
ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []Provider
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (s *SQLStore) UpdateProvider(ctx context.Context, id string, input ProviderUpdate) (Provider, bool, error) {
	provider, ok, err := s.GetProvider(ctx, id)
	if err != nil || !ok {
		return Provider{}, ok, err
	}
	applyProviderUpdate(&provider, input)
	apiKey, _, err := s.APIKey(ctx, id)
	if err != nil {
		return Provider{}, true, err
	}
	if input.APIKey != nil {
		apiKey = *input.APIKey
		provider.HasAPIKey = apiKey != ""
	}
	provider.UpdatedAt = s.now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE ai_providers
SET name = ?, provider_type = ?, base_url = ?, api_key_encrypted = ?, default_model = ?, enabled = ?, updated_at = ?
WHERE id = ?`,
		provider.Name, string(provider.Type), nullString(provider.BaseURL), nullString(apiKey), nullString(provider.DefaultModel),
		boolInt(provider.Enabled), formatTime(provider.UpdatedAt), provider.ID)
	if err != nil {
		return Provider{}, true, err
	}
	return provider, true, nil
}

func (s *SQLStore) DeleteProvider(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM ai_providers WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *SQLStore) APIKey(ctx context.Context, id string) (string, bool, error) {
	var apiKey sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT api_key_encrypted FROM ai_providers WHERE id = ?`, id).Scan(&apiKey)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return apiKey.String, apiKey.Valid && apiKey.String != "", nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProvider(scanner scanner) (Provider, error) {
	var provider Provider
	var providerType string
	var baseURL, apiKey, defaultModel sql.NullString
	var enabled int
	var createdAt, updatedAt string
	if err := scanner.Scan(&provider.ID, &provider.Name, &providerType, &baseURL, &apiKey, &defaultModel, &enabled, &createdAt, &updatedAt); err != nil {
		return Provider{}, err
	}
	provider.Type = ProviderType(providerType)
	provider.BaseURL = baseURL.String
	provider.DefaultModel = defaultModel.String
	provider.Enabled = enabled == 1
	provider.HasAPIKey = apiKey.Valid && apiKey.String != ""
	provider.CreatedAt = parseTime(createdAt)
	provider.UpdatedAt = parseTime(updatedAt)
	return provider, nil
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func nullString(value string) any {
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

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
