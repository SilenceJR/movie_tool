package localization

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

func (s *SQLStore) UpsertMetadata(ctx context.Context, input MetadataInput) (Metadata, error) {
	if err := validateMetadata(input); err != nil {
		return Metadata{}, err
	}
	now := s.now().UTC()
	existing, ok, err := s.getMetadata(ctx, input.MediaID, input.Language, input.FieldName)
	if err != nil {
		return Metadata{}, err
	}
	item := Metadata{
		ID:        newID("localized"),
		MediaID:   input.MediaID,
		Language:  input.Language,
		FieldName: input.FieldName,
		Value:     input.Value,
		Source:    input.Source,
		Provider:  input.Provider,
		Locked:    input.Locked,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if ok {
		item.ID = existing.ID
		item.CreatedAt = existing.CreatedAt
		_, err = s.db.ExecContext(ctx, `
UPDATE localized_metadata SET value = ?, source = ?, provider = ?, locked = ?, updated_at = ?
WHERE id = ?`,
			nullString(item.Value), nullString(item.Source), nullString(item.Provider), boolInt(item.Locked), formatTime(item.UpdatedAt), item.ID)
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO localized_metadata (id, media_id, language, field_name, value, source, provider, locked, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.MediaID, item.Language, item.FieldName, nullString(item.Value), nullString(item.Source), nullString(item.Provider), boolInt(item.Locked), formatTime(item.CreatedAt), formatTime(item.UpdatedAt))
	}
	if err != nil {
		return Metadata{}, err
	}
	return item, nil
}

func (s *SQLStore) ListMetadata(ctx context.Context, query MetadataQuery) ([]Metadata, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 2)
	if query.MediaID != "" {
		where += " AND media_id = ?"
		args = append(args, query.MediaID)
	}
	if query.Language != "" {
		where += " AND language = ?"
		args = append(args, query.Language)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, media_id, language, field_name, value, source, provider, locked, created_at, updated_at
FROM localized_metadata
`+where+`
ORDER BY language ASC, field_name ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Metadata
	for rows.Next() {
		item, err := scanMetadata(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLStore) Translate(ctx context.Context, mediaID string, input TranslateInput) (TranslationCache, *Metadata, error) {
	if err := validateTranslation(mediaID, input); err != nil {
		return TranslationCache{}, nil, err
	}
	now := s.now().UTC()
	hash := TextHash(input.SourceText)
	existing, ok, err := s.getTranslation(ctx, input.SourceLanguage, input.TargetLanguage, hash)
	if err != nil {
		return TranslationCache{}, nil, err
	}
	cache := TranslationCache{
		ID:             newID("translation"),
		SourceLanguage: input.SourceLanguage,
		TargetLanguage: input.TargetLanguage,
		SourceTextHash: hash,
		SourceText:     input.SourceText,
		TranslatedText: input.TranslatedText,
		Provider:       input.Provider,
		Model:          input.Model,
		Status:         statusOrDefault(input.Status),
		Confidence:     input.Confidence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if ok {
		cache.ID = existing.ID
		cache.CreatedAt = existing.CreatedAt
		_, err = s.db.ExecContext(ctx, `
UPDATE translation_cache SET translated_text = ?, provider = ?, model = ?, status = ?, confidence = ?, updated_at = ?
WHERE id = ?`,
			nullString(cache.TranslatedText), nullString(cache.Provider), nullString(cache.Model), cache.Status, nullInt(cache.Confidence), formatTime(cache.UpdatedAt), cache.ID)
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO translation_cache (
  id, source_language, target_language, source_text_hash, source_text, translated_text,
  provider, model, status, confidence, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			cache.ID, cache.SourceLanguage, cache.TargetLanguage, cache.SourceTextHash, cache.SourceText, nullString(cache.TranslatedText),
			nullString(cache.Provider), nullString(cache.Model), cache.Status, nullInt(cache.Confidence), formatTime(cache.CreatedAt), formatTime(cache.UpdatedAt))
	}
	if err != nil {
		return TranslationCache{}, nil, err
	}
	if !input.ApplyToMedia {
		return cache, nil, nil
	}
	metadata, err := s.UpsertMetadata(ctx, MetadataInput{
		MediaID:   mediaID,
		Language:  input.TargetLanguage,
		FieldName: input.FieldName,
		Value:     input.TranslatedText,
		Source:    "translation",
		Provider:  input.Provider,
	})
	if err != nil {
		return TranslationCache{}, nil, err
	}
	return cache, &metadata, nil
}

func (s *SQLStore) ListTranslations(ctx context.Context, _ string) ([]TranslationCache, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, source_language, target_language, source_text_hash, source_text, translated_text,
       provider, model, status, confidence, created_at, updated_at
FROM translation_cache
ORDER BY updated_at DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TranslationCache
	for rows.Next() {
		item, err := scanTranslation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLStore) getMetadata(ctx context.Context, mediaID, language, fieldName string) (Metadata, bool, error) {
	item, err := scanMetadata(s.db.QueryRowContext(ctx, `
SELECT id, media_id, language, field_name, value, source, provider, locked, created_at, updated_at
FROM localized_metadata
WHERE media_id = ? AND language = ? AND field_name = ?`, mediaID, language, fieldName))
	if err == sql.ErrNoRows {
		return Metadata{}, false, nil
	}
	if err != nil {
		return Metadata{}, false, err
	}
	return item, true, nil
}

func (s *SQLStore) getTranslation(ctx context.Context, sourceLanguage, targetLanguage, hash string) (TranslationCache, bool, error) {
	item, err := scanTranslation(s.db.QueryRowContext(ctx, `
SELECT id, source_language, target_language, source_text_hash, source_text, translated_text,
       provider, model, status, confidence, created_at, updated_at
FROM translation_cache
WHERE source_language = ? AND target_language = ? AND source_text_hash = ?`, sourceLanguage, targetLanguage, hash))
	if err == sql.ErrNoRows {
		return TranslationCache{}, false, nil
	}
	if err != nil {
		return TranslationCache{}, false, err
	}
	return item, true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMetadata(scanner scanner) (Metadata, error) {
	var item Metadata
	var value, source, provider sql.NullString
	var locked int
	var createdAt, updatedAt string
	if err := scanner.Scan(&item.ID, &item.MediaID, &item.Language, &item.FieldName, &value, &source, &provider, &locked, &createdAt, &updatedAt); err != nil {
		return Metadata{}, err
	}
	item.Value = value.String
	item.Source = source.String
	item.Provider = provider.String
	item.Locked = locked == 1
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}

func scanTranslation(scanner scanner) (TranslationCache, error) {
	var item TranslationCache
	var translatedText, provider, model sql.NullString
	var confidence sql.NullInt64
	var createdAt, updatedAt string
	if err := scanner.Scan(&item.ID, &item.SourceLanguage, &item.TargetLanguage, &item.SourceTextHash, &item.SourceText, &translatedText, &provider, &model, &item.Status, &confidence, &createdAt, &updatedAt); err != nil {
		return TranslationCache{}, err
	}
	item.TranslatedText = translatedText.String
	item.Provider = provider.String
	item.Model = model.String
	item.Confidence = int(confidence.Int64)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
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

func nullInt(value int) any {
	if value == 0 {
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
