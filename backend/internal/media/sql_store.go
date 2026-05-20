package media

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
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

func (s *SQLStore) UpsertFile(ctx context.Context, input FileInput) (File, error) {
	if err := normalizeFileInput(&input); err != nil {
		return File{}, err
	}

	normalizedPath := normalizePath(input.Path)
	existing, ok, err := s.GetFileByPath(ctx, normalizedPath)
	if err != nil {
		return File{}, err
	}

	now := s.now().UTC()
	file := existing
	if !ok {
		file = File{
			ID:             newID("file"),
			NormalizedPath: normalizedPath,
			CreatedAt:      now,
		}
	}

	file.LibraryID = input.LibraryID
	file.MediaID = input.MediaID
	file.VersionID = input.VersionID
	file.Path = input.Path
	file.FileName = input.FileName
	file.Extension = strings.ToLower(input.Extension)
	file.Size = input.Size
	file.ModifiedAt = input.ModifiedAt
	file.Status = FileStatusAvailable
	file.IsSTRM = input.IsSTRM
	file.STRMTarget = input.STRMTarget
	file.DetectedMediaType = input.DetectedMediaType
	file.ParsedTitle = input.ParsedTitle
	file.ParsedYear = input.ParsedYear
	file.ParsedSeason = input.ParsedSeason
	file.ParsedEpisode = input.ParsedEpisode
	file.ParsedNumber = input.ParsedNumber
	file.FailureError = ""
	file.FailedAt = nil
	file.UpdatedAt = now

	if ok {
		_, err = s.db.ExecContext(ctx, `
UPDATE media_files
SET media_id = ?, version_id = ?, library_id = ?, path = ?, file_name = ?, extension = ?, size = ?, modified_at = ?, file_status = ?,
    is_strm = ?, strm_target = ?, detected_media_type = ?, parsed_title = ?, parsed_year = ?,
    parsed_season = ?, parsed_episode = ?, parsed_number = ?, failure_error = ?, failed_at = ?, updated_at = ?
WHERE normalized_path = ?`,
			nullableString(file.MediaID),
			nullableString(file.VersionID),
			file.LibraryID,
			file.Path,
			file.FileName,
			file.Extension,
			file.Size,
			formatTime(file.ModifiedAt),
			string(file.Status),
			boolInt(file.IsSTRM),
			file.STRMTarget,
			file.DetectedMediaType,
			file.ParsedTitle,
			file.ParsedYear,
			file.ParsedSeason,
			file.ParsedEpisode,
			file.ParsedNumber,
			nullableString(file.FailureError),
			nullableTime(file.FailedAt),
			formatTime(file.UpdatedAt),
			file.NormalizedPath,
		)
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO media_files (
  id, media_id, version_id, library_id, path, normalized_path, file_name, extension, size, modified_at, file_status,
  is_strm, strm_target, detected_media_type, parsed_title, parsed_year, parsed_season,
  parsed_episode, parsed_number, failure_error, failed_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			file.ID,
			nullableString(file.MediaID),
			nullableString(file.VersionID),
			file.LibraryID,
			file.Path,
			file.NormalizedPath,
			file.FileName,
			file.Extension,
			file.Size,
			formatTime(file.ModifiedAt),
			string(file.Status),
			boolInt(file.IsSTRM),
			file.STRMTarget,
			file.DetectedMediaType,
			file.ParsedTitle,
			file.ParsedYear,
			file.ParsedSeason,
			file.ParsedEpisode,
			file.ParsedNumber,
			nullableString(file.FailureError),
			nullableTime(file.FailedAt),
			formatTime(file.CreatedAt),
			formatTime(file.UpdatedAt),
		)
	}
	if err != nil {
		return File{}, err
	}
	return file, nil
}

func (s *SQLStore) MarkFileFailed(ctx context.Context, input FailedFileInput) (File, error) {
	if err := normalizeFailedFileInput(&input); err != nil {
		return File{}, err
	}

	normalizedPath := normalizePath(input.Path)
	existing, ok, err := s.GetFileByPath(ctx, normalizedPath)
	if err != nil {
		return File{}, err
	}

	now := s.now().UTC()
	file := existing
	if !ok {
		file = File{
			ID:             newID("file"),
			NormalizedPath: normalizedPath,
			CreatedAt:      now,
		}
	}

	file.LibraryID = input.LibraryID
	file.MediaID = ""
	file.VersionID = ""
	file.Path = input.Path
	file.FileName = input.FileName
	file.Extension = strings.ToLower(input.Extension)
	file.Size = input.Size
	file.ModifiedAt = input.ModifiedAt
	file.Status = FileStatusFailed
	file.IsSTRM = false
	file.STRMTarget = ""
	file.DetectedMediaType = input.DetectedMediaType
	file.ParsedTitle = input.ParsedTitle
	file.ParsedYear = input.ParsedYear
	file.ParsedSeason = input.ParsedSeason
	file.ParsedEpisode = input.ParsedEpisode
	file.ParsedNumber = input.ParsedNumber
	file.FailureError = input.Error
	file.FailedAt = &now
	file.UpdatedAt = now

	if ok {
		_, err = s.db.ExecContext(ctx, `
UPDATE media_files
SET media_id = NULL, version_id = NULL, library_id = ?, path = ?, file_name = ?, extension = ?, size = ?, modified_at = ?, file_status = ?,
    is_strm = 0, strm_target = NULL, detected_media_type = ?, parsed_title = ?, parsed_year = ?,
    parsed_season = ?, parsed_episode = ?, parsed_number = ?, failure_error = ?, failed_at = ?, updated_at = ?
WHERE normalized_path = ?`,
			file.LibraryID,
			file.Path,
			file.FileName,
			file.Extension,
			file.Size,
			formatTime(file.ModifiedAt),
			string(file.Status),
			file.DetectedMediaType,
			file.ParsedTitle,
			file.ParsedYear,
			file.ParsedSeason,
			file.ParsedEpisode,
			file.ParsedNumber,
			file.FailureError,
			formatTime(*file.FailedAt),
			formatTime(file.UpdatedAt),
			file.NormalizedPath,
		)
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO media_files (
  id, media_id, version_id, library_id, path, normalized_path, file_name, extension, size, modified_at, file_status,
  is_strm, strm_target, detected_media_type, parsed_title, parsed_year, parsed_season,
  parsed_episode, parsed_number, failure_error, failed_at, created_at, updated_at
) VALUES (?, NULL, NULL, ?, ?, ?, ?, ?, ?, ?, ?, 0, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			file.ID,
			file.LibraryID,
			file.Path,
			file.NormalizedPath,
			file.FileName,
			file.Extension,
			file.Size,
			formatTime(file.ModifiedAt),
			string(file.Status),
			file.DetectedMediaType,
			file.ParsedTitle,
			file.ParsedYear,
			file.ParsedSeason,
			file.ParsedEpisode,
			file.ParsedNumber,
			file.FailureError,
			formatTime(*file.FailedAt),
			formatTime(file.CreatedAt),
			formatTime(file.UpdatedAt),
		)
	}
	if err != nil {
		return File{}, err
	}
	return file, nil
}

func (s *SQLStore) ListFilesByLibrary(ctx context.Context, libraryID string) ([]File, error) {
	return s.ListFiles(ctx, FileQuery{LibraryID: libraryID})
}

func (s *SQLStore) GetFile(ctx context.Context, id string) (File, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, media_id, version_id, library_id, path, normalized_path, file_name, extension, size, modified_at,
       file_status, is_strm, strm_target, detected_media_type, parsed_title, parsed_year,
       parsed_season, parsed_episode, parsed_number, failure_error, failed_at, created_at, updated_at
FROM media_files
WHERE id = ?`, id)

	file, err := scanFile(row)
	if err == sql.ErrNoRows {
		return File{}, false, nil
	}
	if err != nil {
		return File{}, false, err
	}
	return file, true, nil
}

func (s *SQLStore) UpdateFilePath(ctx context.Context, id string, path string) (File, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return File{}, false, fmt.Errorf("file path is required")
	}
	file, ok, err := s.GetFile(ctx, id)
	if err != nil || !ok {
		return File{}, ok, err
	}
	now := s.now().UTC()
	file.Path = path
	file.NormalizedPath = normalizePath(path)
	file.FileName = filepath.Base(path)
	file.Extension = strings.ToLower(filepath.Ext(path))
	file.Status = FileStatusAvailable
	file.FailureError = ""
	file.FailedAt = nil
	file.UpdatedAt = now

	_, err = s.db.ExecContext(ctx, `
UPDATE media_files
SET path = ?, normalized_path = ?, file_name = ?, extension = ?, file_status = ?, failure_error = NULL, failed_at = NULL, updated_at = ?
WHERE id = ?`,
		file.Path,
		file.NormalizedPath,
		file.FileName,
		file.Extension,
		string(file.Status),
		formatTime(file.UpdatedAt),
		file.ID,
	)
	if err != nil {
		return File{}, true, err
	}
	return file, true, nil
}

func (s *SQLStore) ListFiles(ctx context.Context, query FileQuery) ([]File, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 8)
	if query.LibraryID != "" {
		where += " AND library_id = ?"
		args = append(args, query.LibraryID)
	}
	if query.MediaID != "" {
		where += " AND media_id = ?"
		args = append(args, query.MediaID)
	}
	if query.Status != "" {
		where += " AND file_status = ?"
		args = append(args, string(query.Status))
	}
	if query.PathPrefix != "" {
		where += " AND normalized_path LIKE ?"
		args = append(args, normalizePath(query.PathPrefix)+"%")
	}
	if query.DetectedMediaType != "" {
		where += " AND detected_media_type = ?"
		args = append(args, query.DetectedMediaType)
	}
	if query.FailureContains != "" {
		where += " AND lower(failure_error) LIKE ?"
		args = append(args, "%"+strings.ToLower(query.FailureContains)+"%")
	}
	if query.FailedAfter != nil {
		where += " AND failed_at >= ?"
		args = append(args, query.FailedAfter.UTC().Format(time.RFC3339Nano))
	}
	if query.FailedBefore != nil {
		where += " AND failed_at <= ?"
		args = append(args, query.FailedBefore.UTC().Format(time.RFC3339Nano))
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, media_id, version_id, library_id, path, normalized_path, file_name, extension, size, modified_at,
       file_status, is_strm, strm_target, detected_media_type, parsed_title, parsed_year,
       parsed_season, parsed_episode, parsed_number, failure_error, failed_at, created_at, updated_at
FROM media_files
`+where+`
ORDER BY path ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func (s *SQLStore) GetFileByPath(ctx context.Context, path string) (File, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, media_id, version_id, library_id, path, normalized_path, file_name, extension, size, modified_at,
       file_status, is_strm, strm_target, detected_media_type, parsed_title, parsed_year,
       parsed_season, parsed_episode, parsed_number, failure_error, failed_at, created_at, updated_at
FROM media_files
WHERE normalized_path = ?`, normalizePath(path))

	file, err := scanFile(row)
	if err == sql.ErrNoRows {
		return File{}, false, nil
	}
	if err != nil {
		return File{}, false, err
	}
	return file, true, nil
}

func (s *SQLStore) MarkMissingByLibrary(ctx context.Context, libraryID string, availablePaths []string) (int, error) {
	files, err := s.ListFilesByLibrary(ctx, libraryID)
	if err != nil {
		return 0, err
	}

	available := make(map[string]struct{}, len(availablePaths))
	for _, path := range availablePaths {
		available[normalizePath(path)] = struct{}{}
	}

	changed := 0
	now := s.now().UTC()
	for _, file := range files {
		if _, ok := available[file.NormalizedPath]; ok {
			continue
		}
		if file.Status == FileStatusMissing {
			continue
		}
		_, err := s.db.ExecContext(ctx, `
UPDATE media_files
SET file_status = ?, updated_at = ?
WHERE id = ?`,
			string(FileStatusMissing),
			formatTime(now),
			file.ID,
		)
		if err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

func (s *SQLStore) DeleteMissingByLibrary(ctx context.Context, libraryID string) (int, error) {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM media_files
WHERE library_id = ? AND file_status = ?`,
		libraryID,
		string(FileStatusMissing),
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

type fileScanner interface {
	Scan(dest ...any) error
}

func scanFile(scanner fileScanner) (File, error) {
	var file File
	var mediaID sql.NullString
	var versionID sql.NullString
	var modifiedAt string
	var createdAt string
	var updatedAt string
	var status string
	var isSTRM int
	var strmTarget sql.NullString
	var failureError sql.NullString
	var failedAt sql.NullString

	err := scanner.Scan(
		&file.ID,
		&mediaID,
		&versionID,
		&file.LibraryID,
		&file.Path,
		&file.NormalizedPath,
		&file.FileName,
		&file.Extension,
		&file.Size,
		&modifiedAt,
		&status,
		&isSTRM,
		&strmTarget,
		&file.DetectedMediaType,
		&file.ParsedTitle,
		&file.ParsedYear,
		&file.ParsedSeason,
		&file.ParsedEpisode,
		&file.ParsedNumber,
		&failureError,
		&failedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return File{}, err
	}

	file.MediaID = mediaID.String
	file.VersionID = versionID.String
	file.Status = FileStatus(status)
	file.IsSTRM = isSTRM == 1
	file.STRMTarget = strmTarget.String
	file.FailureError = failureError.String
	if failedAt.Valid && failedAt.String != "" {
		parsed := parseTime(failedAt.String)
		file.FailedAt = &parsed
	}
	file.ModifiedAt = parseTime(modifiedAt)
	file.CreatedAt = parseTime(createdAt)
	file.UpdatedAt = parseTime(updatedAt)
	return file, nil
}

func normalizeFileInput(input *FileInput) error {
	input.Path = strings.TrimSpace(input.Path)
	input.LibraryID = strings.TrimSpace(input.LibraryID)
	if input.LibraryID == "" {
		return fmt.Errorf("library id is required")
	}
	if input.Path == "" {
		return fmt.Errorf("file path is required")
	}
	if input.FileName == "" {
		input.FileName = filepath.Base(input.Path)
	}
	if input.Extension == "" {
		input.Extension = strings.ToLower(filepath.Ext(input.Path))
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

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return formatTime(*value)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
