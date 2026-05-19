package integration

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

func (s *SQLStore) Create(ctx context.Context, input ServerInput) (Server, error) {
	server, err := serverFromInput(input, s.now().UTC())
	if err != nil {
		return Server{}, err
	}
	server.ID = newID("integration")
	server.HasAPIKey = input.APIKey != ""
	_, err = s.db.ExecContext(ctx, `
INSERT INTO server_integrations (id, name, server_type, base_url, api_key_encrypted, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		server.ID, server.Name, string(server.Type), server.BaseURL, nullString(input.APIKey), boolInt(server.Enabled), formatTime(server.CreatedAt), formatTime(server.UpdatedAt))
	if err != nil {
		return Server{}, err
	}
	return server, nil
}

func (s *SQLStore) Get(ctx context.Context, id string) (Server, bool, error) {
	server, err := scanServer(s.db.QueryRowContext(ctx, `
SELECT id, name, server_type, base_url, api_key_encrypted, enabled, created_at, updated_at
FROM server_integrations
WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Server{}, false, nil
	}
	if err != nil {
		return Server{}, false, err
	}
	return server, true, nil
}

func (s *SQLStore) List(ctx context.Context) ([]Server, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, server_type, base_url, api_key_encrypted, enabled, created_at, updated_at
FROM server_integrations
ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []Server
	for rows.Next() {
		server, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

func (s *SQLStore) Update(ctx context.Context, id string, input ServerUpdate) (Server, bool, error) {
	server, ok, err := s.Get(ctx, id)
	if err != nil || !ok {
		return Server{}, ok, err
	}
	applyUpdate(&server, input)
	if err := validateServer(server.Name, server.Type, server.BaseURL); err != nil {
		return Server{}, true, err
	}
	apiKey, _, err := s.APIKey(ctx, id)
	if err != nil {
		return Server{}, true, err
	}
	if input.APIKey != nil {
		apiKey = *input.APIKey
		server.HasAPIKey = apiKey != ""
	}
	server.UpdatedAt = s.now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE server_integrations
SET name = ?, server_type = ?, base_url = ?, api_key_encrypted = ?, enabled = ?, updated_at = ?
WHERE id = ?`,
		server.Name, string(server.Type), server.BaseURL, nullString(apiKey), boolInt(server.Enabled), formatTime(server.UpdatedAt), server.ID)
	if err != nil {
		return Server{}, true, err
	}
	return server, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM server_integrations WHERE id = ?`, id)
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
	err := s.db.QueryRowContext(ctx, `SELECT api_key_encrypted FROM server_integrations WHERE id = ?`, id).Scan(&apiKey)
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

func scanServer(scanner scanner) (Server, error) {
	var server Server
	var serverType string
	var apiKey sql.NullString
	var enabled int
	var createdAt, updatedAt string
	if err := scanner.Scan(&server.ID, &server.Name, &serverType, &server.BaseURL, &apiKey, &enabled, &createdAt, &updatedAt); err != nil {
		return Server{}, err
	}
	server.Type = ServerType(serverType)
	server.HasAPIKey = apiKey.Valid && apiKey.String != ""
	server.Enabled = enabled == 1
	server.CreatedAt = parseTime(createdAt)
	server.UpdatedAt = parseTime(updatedAt)
	return server, nil
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
