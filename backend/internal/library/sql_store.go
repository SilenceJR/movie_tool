package library

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
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

func (s *SQLStore) List(ctx context.Context) ([]Library, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, media_type, path, language, cache_policy, nfo_enabled, strm_enabled, watch_enabled, created_at, updated_at
FROM libraries
ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []Library
	for rows.Next() {
		library, err := scanLibrary(rows)
		if err != nil {
			return nil, err
		}
		libraries = append(libraries, library)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return libraries, nil
}

func (s *SQLStore) Get(ctx context.Context, id string) (Library, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, media_type, path, language, cache_policy, nfo_enabled, strm_enabled, watch_enabled, created_at, updated_at
FROM libraries
WHERE id = ?`, id)
	library, err := scanLibrary(row)
	if err == sql.ErrNoRows {
		return Library{}, false, nil
	}
	if err != nil {
		return Library{}, false, err
	}
	return library, true, nil
}

func (s *SQLStore) Create(ctx context.Context, input CreateInput) (Library, error) {
	if err := normalizeCreateInput(&input); err != nil {
		return Library{}, err
	}

	now := s.now().UTC()
	library := Library{
		ID:           newID("lib"),
		Name:         input.Name,
		MediaType:    input.MediaType,
		Path:         input.Path,
		Language:     input.Language,
		CachePolicy:  input.CachePolicy,
		NFOEnabled:   input.NFOEnabled,
		STRMEnabled:  input.STRMEnabled,
		WatchEnabled: input.WatchEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO libraries (
  id, name, media_type, path, language, cache_policy, nfo_enabled, strm_enabled, watch_enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		library.ID,
		library.Name,
		string(library.MediaType),
		library.Path,
		library.Language,
		library.CachePolicy,
		boolInt(library.NFOEnabled),
		boolInt(library.STRMEnabled),
		boolInt(library.WatchEnabled),
		formatTime(library.CreatedAt),
		formatTime(library.UpdatedAt),
	)
	if err != nil {
		return Library{}, err
	}
	return library, nil
}

func (s *SQLStore) Update(ctx context.Context, id string, input UpdateInput) (Library, bool, error) {
	found, ok, err := s.Get(ctx, id)
	if err != nil || !ok {
		return Library{}, ok, err
	}

	if input.Name != nil {
		found.Name = strings.TrimSpace(*input.Name)
	}
	if input.MediaType != nil {
		found.MediaType = *input.MediaType
	}
	if input.Path != nil {
		found.Path = strings.TrimSpace(*input.Path)
	}
	if input.Language != nil {
		found.Language = strings.TrimSpace(*input.Language)
	}
	if input.CachePolicy != nil {
		found.CachePolicy = strings.TrimSpace(*input.CachePolicy)
	}
	if input.NFOEnabled != nil {
		found.NFOEnabled = *input.NFOEnabled
	}
	if input.STRMEnabled != nil {
		found.STRMEnabled = *input.STRMEnabled
	}
	if input.WatchEnabled != nil {
		found.WatchEnabled = *input.WatchEnabled
	}
	if found.Name == "" {
		return Library{}, true, fmt.Errorf("library name is required")
	}
	if found.Path == "" {
		return Library{}, true, fmt.Errorf("library path is required")
	}
	found.UpdatedAt = s.now().UTC()

	_, err = s.db.ExecContext(ctx, `
UPDATE libraries
SET name = ?, media_type = ?, path = ?, language = ?, cache_policy = ?, nfo_enabled = ?, strm_enabled = ?, watch_enabled = ?, updated_at = ?
WHERE id = ?`,
		found.Name,
		string(found.MediaType),
		found.Path,
		found.Language,
		found.CachePolicy,
		boolInt(found.NFOEnabled),
		boolInt(found.STRMEnabled),
		boolInt(found.WatchEnabled),
		formatTime(found.UpdatedAt),
		found.ID,
	)
	if err != nil {
		return Library{}, true, err
	}
	return found, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM libraries WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type libraryScanner interface {
	Scan(dest ...any) error
}

func scanLibrary(scanner libraryScanner) (Library, error) {
	var library Library
	var mediaType string
	var nfoEnabled int
	var strmEnabled int
	var watchEnabled int
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&library.ID,
		&library.Name,
		&mediaType,
		&library.Path,
		&library.Language,
		&library.CachePolicy,
		&nfoEnabled,
		&strmEnabled,
		&watchEnabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Library{}, err
	}

	library.MediaType = MediaType(mediaType)
	library.NFOEnabled = nfoEnabled == 1
	library.STRMEnabled = strmEnabled == 1
	library.WatchEnabled = watchEnabled == 1
	library.CreatedAt = parseTime(createdAt)
	library.UpdatedAt = parseTime(updatedAt)
	return library, nil
}

func normalizeCreateInput(input *CreateInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Path = strings.TrimSpace(input.Path)
	input.Language = strings.TrimSpace(input.Language)
	input.CachePolicy = strings.TrimSpace(input.CachePolicy)
	if input.Name == "" {
		return fmt.Errorf("library name is required")
	}
	if input.Path == "" {
		return fmt.Errorf("library path is required")
	}
	if input.MediaType == "" {
		input.MediaType = MediaTypeOther
	}
	if input.Language == "" {
		input.Language = "zh-CN"
	}
	if input.CachePolicy == "" {
		input.CachePolicy = "global"
	}
	return nil
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
