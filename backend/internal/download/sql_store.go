package download

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

func (s *SQLStore) Create(ctx context.Context, input DirectoryInput) (Directory, error) {
	directory, err := directoryFromInput(input, s.now().UTC())
	if err != nil {
		return Directory{}, err
	}
	directory.ID = newID("download_dir")
	_, err = s.db.ExecContext(ctx, `
INSERT INTO download_directories (id, name, path, library_id, media_type, action_mode, enabled, watch_enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		directory.ID, directory.Name, directory.Path, directory.LibraryID, nullString(directory.MediaType), directory.ActionMode,
		boolInt(directory.Enabled), boolInt(directory.WatchEnabled), formatTime(directory.CreatedAt), formatTime(directory.UpdatedAt))
	if err != nil {
		return Directory{}, err
	}
	return directory, nil
}

func (s *SQLStore) Get(ctx context.Context, id string) (Directory, bool, error) {
	directory, err := scanDirectory(s.db.QueryRowContext(ctx, `
SELECT id, name, path, library_id, media_type, action_mode, enabled, watch_enabled, created_at, updated_at
FROM download_directories
WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Directory{}, false, nil
	}
	if err != nil {
		return Directory{}, false, err
	}
	return directory, true, nil
}

func (s *SQLStore) List(ctx context.Context) ([]Directory, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, path, library_id, media_type, action_mode, enabled, watch_enabled, created_at, updated_at
FROM download_directories
ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var directories []Directory
	for rows.Next() {
		directory, err := scanDirectory(rows)
		if err != nil {
			return nil, err
		}
		directories = append(directories, directory)
	}
	return directories, rows.Err()
}

func (s *SQLStore) Update(ctx context.Context, id string, input DirectoryUpdate) (Directory, bool, error) {
	directory, ok, err := s.Get(ctx, id)
	if err != nil || !ok {
		return Directory{}, ok, err
	}
	applyUpdate(&directory, input)
	if err := validate(directory); err != nil {
		return Directory{}, true, err
	}
	directory.UpdatedAt = s.now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE download_directories
SET name = ?, path = ?, library_id = ?, media_type = ?, action_mode = ?, enabled = ?, watch_enabled = ?, updated_at = ?
WHERE id = ?`,
		directory.Name, directory.Path, directory.LibraryID, nullString(directory.MediaType), directory.ActionMode,
		boolInt(directory.Enabled), boolInt(directory.WatchEnabled), formatTime(directory.UpdatedAt), directory.ID)
	if err != nil {
		return Directory{}, true, err
	}
	return directory, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM download_directories WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type directoryScanner interface {
	Scan(dest ...any) error
}

func scanDirectory(scanner directoryScanner) (Directory, error) {
	var directory Directory
	var mediaType sql.NullString
	var enabled int
	var watchEnabled int
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(&directory.ID, &directory.Name, &directory.Path, &directory.LibraryID, &mediaType, &directory.ActionMode, &enabled, &watchEnabled, &createdAt, &updatedAt); err != nil {
		return Directory{}, err
	}
	directory.MediaType = mediaType.String
	directory.Enabled = enabled == 1
	directory.WatchEnabled = watchEnabled == 1
	directory.CreatedAt = parseTime(createdAt)
	directory.UpdatedAt = parseTime(updatedAt)
	return directory, nil
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

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
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
