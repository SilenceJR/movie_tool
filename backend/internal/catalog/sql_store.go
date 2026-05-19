package catalog

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

func (s *SQLStore) CreateItem(ctx context.Context, input ItemInput) (Item, error) {
	item, err := itemFromInput(input, s.now().UTC())
	if err != nil {
		return Item{}, err
	}
	item.ID = newID("media")
	_, err = s.db.ExecContext(ctx, `
INSERT INTO media_items (
  id, library_id, media_type, title, original_title, display_title, year, overview,
  original_language, display_language, release_date, runtime, status, match_status,
  locked, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.LibraryID, item.MediaType, nullString(item.Title), nullString(item.OriginalTitle),
		nullString(item.DisplayTitle), nullInt(item.Year), nullString(item.Overview), nullString(item.OriginalLang),
		item.DisplayLang, nullString(item.ReleaseDate), nullInt(item.Runtime), item.Status, string(item.MatchStatus),
		boolInt(item.Locked), formatTime(item.CreatedAt), formatTime(item.UpdatedAt),
	)
	if err != nil {
		return Item{}, err
	}
	return item, nil
}

func (s *SQLStore) GetItem(ctx context.Context, id string) (Item, bool, error) {
	row := s.db.QueryRowContext(ctx, itemSelect()+`
WHERE media_items.id = ?
GROUP BY media_items.id`, id)
	item, err := scanItem(row)
	if err == sql.ErrNoRows {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, err
	}
	return item, true, nil
}

func (s *SQLStore) ListItems(ctx context.Context, query ItemQuery) ([]Item, error) {
	sqlQuery, args := listItemsSQL(query)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStore) UpdateItem(ctx context.Context, id string, input ItemUpdate) (Item, bool, error) {
	item, ok, err := s.GetItem(ctx, id)
	if err != nil || !ok {
		return Item{}, ok, err
	}
	applyItemUpdate(&item, input)
	item.UpdatedAt = s.now().UTC()
	_, err = s.db.ExecContext(ctx, `
UPDATE media_items SET
  title = ?, original_title = ?, display_title = ?, year = ?, overview = ?,
  original_language = ?, display_language = ?, release_date = ?, runtime = ?,
  status = ?, match_status = ?, locked = ?, updated_at = ?
WHERE id = ?`,
		nullString(item.Title), nullString(item.OriginalTitle), nullString(item.DisplayTitle), nullInt(item.Year),
		nullString(item.Overview), nullString(item.OriginalLang), item.DisplayLang, nullString(item.ReleaseDate),
		nullInt(item.Runtime), item.Status, string(item.MatchStatus), boolInt(item.Locked), formatTime(item.UpdatedAt), item.ID,
	)
	if err != nil {
		return Item{}, true, err
	}
	return item, true, nil
}

func (s *SQLStore) CreateVersion(ctx context.Context, mediaID string, input VersionInput) (Version, error) {
	if mediaID == "" {
		return Version{}, fmt.Errorf("media id is required")
	}
	now := s.now().UTC()
	version := Version{
		ID:             newID("version"),
		MediaID:        mediaID,
		Name:           input.Name,
		Resolution:     input.Resolution,
		Source:         input.Source,
		VideoCodec:     input.VideoCodec,
		AudioCodec:     input.AudioCodec,
		HDRFormat:      input.HDRFormat,
		Edition:        input.Edition,
		ReleaseGroup:   input.ReleaseGroup,
		AudioLanguages: input.AudioLanguages,
		SubtitleFlags:  input.SubtitleFlags,
		QualityScore:   input.QualityScore,
		IsDefault:      input.IsDefault,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if version.IsDefault {
		if _, err := s.db.ExecContext(ctx, `UPDATE media_versions SET is_default = 0, updated_at = ? WHERE media_id = ?`, formatTime(now), mediaID); err != nil {
			return Version{}, err
		}
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO media_versions (
  id, media_id, name, resolution, source, video_codec, audio_codec, hdr_format,
  edition, release_group, audio_languages, subtitle_flags, quality_score,
  is_default, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		version.ID, version.MediaID, nullString(version.Name), nullString(version.Resolution), nullString(version.Source),
		nullString(version.VideoCodec), nullString(version.AudioCodec), nullString(version.HDRFormat),
		nullString(version.Edition), nullString(version.ReleaseGroup), nullString(version.AudioLanguages),
		nullString(version.SubtitleFlags), nullInt(version.QualityScore), boolInt(version.IsDefault),
		formatTime(version.CreatedAt), formatTime(version.UpdatedAt),
	)
	if err != nil {
		return Version{}, err
	}
	return version, nil
}

func (s *SQLStore) ListVersions(ctx context.Context, mediaID string) ([]Version, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, media_id, name, resolution, source, video_codec, audio_codec, hdr_format,
       edition, release_group, audio_languages, subtitle_flags, quality_score,
       is_default, created_at, updated_at
FROM media_versions
WHERE media_id = ?
ORDER BY is_default DESC, quality_score DESC, created_at ASC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []Version
	for rows.Next() {
		version, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

func (s *SQLStore) SetDefaultVersion(ctx context.Context, mediaID, versionID string) (Version, bool, error) {
	versions, err := s.ListVersions(ctx, mediaID)
	if err != nil {
		return Version{}, false, err
	}
	var selected Version
	found := false
	for _, version := range versions {
		if version.ID == versionID {
			selected = version
			found = true
			break
		}
	}
	if !found {
		return Version{}, false, nil
	}
	now := s.now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE media_versions SET is_default = 0, updated_at = ? WHERE media_id = ?`, formatTime(now), mediaID); err != nil {
		return Version{}, true, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE media_versions SET is_default = 1, updated_at = ? WHERE id = ?`, formatTime(now), versionID); err != nil {
		return Version{}, true, err
	}
	selected.IsDefault = true
	selected.UpdatedAt = now
	return selected, true, nil
}

func (s *SQLStore) CreatePerson(ctx context.Context, input PersonInput) (Person, error) {
	if input.Name == "" {
		return Person{}, fmt.Errorf("person name is required")
	}
	now := s.now().UTC()
	person := Person{ID: newID("person"), Name: input.Name, OriginalName: input.OriginalName, LocalizedName: input.LocalizedName, Gender: input.Gender, Avatar: input.Avatar, Bio: input.Bio, BirthDate: input.BirthDate, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO people (id, name, original_name, localized_name, gender, avatar, bio, birth_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		person.ID, person.Name, nullString(person.OriginalName), nullString(person.LocalizedName), nullString(person.Gender), nullString(person.Avatar), nullString(person.Bio), nullString(person.BirthDate), formatTime(person.CreatedAt), formatTime(person.UpdatedAt))
	if err != nil {
		return Person{}, err
	}
	return person, nil
}

func (s *SQLStore) GetPerson(ctx context.Context, id string) (Person, bool, error) {
	person, err := scanPerson(s.db.QueryRowContext(ctx, `SELECT id, name, original_name, localized_name, gender, avatar, bio, birth_date, created_at, updated_at FROM people WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Person{}, false, nil
	}
	if err != nil {
		return Person{}, false, err
	}
	return person, true, nil
}

func (s *SQLStore) ListPeople(ctx context.Context) ([]Person, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, original_name, localized_name, gender, avatar, bio, birth_date, created_at, updated_at FROM people ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var people []Person
	for rows.Next() {
		person, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}

func (s *SQLStore) AddMediaPerson(ctx context.Context, mediaID string, input MediaPersonInput) error {
	if mediaID == "" || input.PersonID == "" {
		return fmt.Errorf("media id and person id are required")
	}
	if input.Role == "" {
		input.Role = "actor"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO media_people (id, media_id, person_id, role, character_name, sort_order)
VALUES (?, ?, ?, ?, ?, ?)`,
		newID("media_person"), mediaID, input.PersonID, input.Role, nullString(input.CharacterName), nullInt(input.SortOrder))
	return err
}

func (s *SQLStore) CreateTag(ctx context.Context, input TagInput) (Tag, error) {
	if input.Name == "" {
		return Tag{}, fmt.Errorf("tag name is required")
	}
	if input.NormalizedName == "" {
		input.NormalizedName = normalizeName(input.Name)
	}
	now := s.now().UTC()
	tag := Tag{ID: newID("tag"), Name: input.Name, NormalizedName: input.NormalizedName, Category: input.Category, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tags (id, name, normalized_name, category, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		tag.ID, tag.Name, tag.NormalizedName, nullString(tag.Category), formatTime(tag.CreatedAt), formatTime(tag.UpdatedAt))
	if err != nil {
		return Tag{}, err
	}
	return tag, nil
}

func (s *SQLStore) GetTag(ctx context.Context, id string) (Tag, bool, error) {
	tag, err := scanTag(s.db.QueryRowContext(ctx, `SELECT id, name, normalized_name, category, created_at, updated_at FROM tags WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Tag{}, false, nil
	}
	if err != nil {
		return Tag{}, false, err
	}
	return tag, true, nil
}

func (s *SQLStore) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, normalized_name, category, created_at, updated_at FROM tags ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []Tag
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *SQLStore) AddMediaTag(ctx context.Context, mediaID string, input MediaTagInput) error {
	if mediaID == "" || input.TagID == "" {
		return fmt.Errorf("media id and tag id are required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO media_tags (id, media_id, tag_id, source)
VALUES (?, ?, ?, ?)`,
		newID("media_tag"), mediaID, input.TagID, nullString(input.Source))
	return err
}

func (s *SQLStore) CreateCollection(ctx context.Context, input CollectionInput) (Collection, error) {
	if input.Name == "" {
		return Collection{}, fmt.Errorf("collection name is required")
	}
	if input.Type == "" {
		input.Type = "manual"
	}
	now := s.now().UTC()
	collection := Collection{ID: newID("collection"), Name: input.Name, LocalizedName: input.LocalizedName, Type: input.Type, Description: input.Description, Poster: input.Poster, Source: input.Source, ExternalID: input.ExternalID, Locked: input.Locked, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO collections (id, name, localized_name, type, description, poster, source, external_id, locked, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		collection.ID, collection.Name, nullString(collection.LocalizedName), collection.Type, nullString(collection.Description), nullString(collection.Poster), nullString(collection.Source), nullString(collection.ExternalID), boolInt(collection.Locked), formatTime(collection.CreatedAt), formatTime(collection.UpdatedAt))
	if err != nil {
		return Collection{}, err
	}
	return collection, nil
}

func (s *SQLStore) GetCollection(ctx context.Context, id string) (Collection, bool, error) {
	collection, err := scanCollection(s.db.QueryRowContext(ctx, `SELECT id, name, localized_name, type, description, poster, source, external_id, locked, created_at, updated_at FROM collections WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Collection{}, false, nil
	}
	if err != nil {
		return Collection{}, false, err
	}
	return collection, true, nil
}

func (s *SQLStore) ListCollections(ctx context.Context) ([]Collection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, localized_name, type, description, poster, source, external_id, locked, created_at, updated_at FROM collections ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var collections []Collection
	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	return collections, rows.Err()
}

func (s *SQLStore) AddCollectionItem(ctx context.Context, collectionID string, input CollectionItemInput) error {
	if collectionID == "" || input.MediaID == "" {
		return fmt.Errorf("collection id and media id are required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO collection_items (id, collection_id, media_id, sort_order, relation_type)
VALUES (?, ?, ?, ?, ?)`,
		newID("collection_item"), collectionID, input.MediaID, nullInt(input.SortOrder), nullString(input.RelationType))
	return err
}

func (s *SQLStore) UpsertExternalID(ctx context.Context, input ExternalIDInput) (ExternalID, error) {
	externalID, err := externalIDFromInput(input, s.now().UTC())
	if err != nil {
		return ExternalID{}, err
	}
	externalID.ID = newID("external_id")
	_, err = s.db.ExecContext(ctx, `
INSERT INTO external_ids (id, entity_type, entity_id, provider, external_id, url, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(entity_type, entity_id, provider) DO UPDATE SET
  external_id = excluded.external_id,
  url = excluded.url`,
		externalID.ID, externalID.EntityType, externalID.EntityID, externalID.Provider, externalID.ExternalID, nullString(externalID.URL), formatTime(externalID.CreatedAt))
	if err != nil {
		return ExternalID{}, err
	}
	stored, ok, err := s.getExternalID(ctx, input.EntityType, input.EntityID, input.Provider)
	if err != nil {
		return ExternalID{}, err
	}
	if !ok {
		return ExternalID{}, fmt.Errorf("external id was not stored")
	}
	return stored, nil
}

func (s *SQLStore) ListExternalIDs(ctx context.Context, entityType, entityID string) ([]ExternalID, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, entity_type, entity_id, provider, external_id, url, created_at
FROM external_ids
WHERE entity_type = ? AND entity_id = ?
ORDER BY provider ASC, external_id ASC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var externalIDs []ExternalID
	for rows.Next() {
		externalID, err := scanExternalID(rows)
		if err != nil {
			return nil, err
		}
		externalIDs = append(externalIDs, externalID)
	}
	return externalIDs, rows.Err()
}

func (s *SQLStore) getExternalID(ctx context.Context, entityType, entityID, provider string) (ExternalID, bool, error) {
	externalID, err := scanExternalID(s.db.QueryRowContext(ctx, `
SELECT id, entity_type, entity_id, provider, external_id, url, created_at
FROM external_ids
WHERE entity_type = ? AND entity_id = ? AND provider = ?`, entityType, entityID, provider))
	if err == sql.ErrNoRows {
		return ExternalID{}, false, nil
	}
	if err != nil {
		return ExternalID{}, false, err
	}
	return externalID, true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func itemSelect() string {
	return `
SELECT media_items.id, media_items.library_id, media_items.media_type, media_items.title,
       media_items.original_title, media_items.display_title, media_items.year, media_items.overview,
       media_items.original_language, media_items.display_language, media_items.release_date,
       media_items.runtime, media_items.status, media_items.match_status, media_items.locked,
       media_items.created_at, media_items.updated_at,
       COUNT(DISTINCT media_versions.id) AS version_count,
       SUM(CASE WHEN media_files.file_status = 'available' THEN 1 ELSE 0 END) AS available_files,
       SUM(CASE WHEN media_files.file_status = 'missing' THEN 1 ELSE 0 END) AS missing_files
FROM media_items
LEFT JOIN media_versions ON media_versions.media_id = media_items.id
LEFT JOIN media_files ON media_files.media_id = media_items.id
`
}

func listItemsSQL(query ItemQuery) (string, []any) {
	joins := ""
	where := "WHERE 1 = 1"
	args := make([]any, 0, 8)
	if query.PersonID != "" {
		joins += " INNER JOIN media_people ON media_people.media_id = media_items.id"
		where += " AND media_people.person_id = ?"
		args = append(args, query.PersonID)
	}
	if query.TagID != "" {
		joins += " INNER JOIN media_tags ON media_tags.media_id = media_items.id"
		where += " AND media_tags.tag_id = ?"
		args = append(args, query.TagID)
	}
	if query.CollectionID != "" {
		joins += " INNER JOIN collection_items ON collection_items.media_id = media_items.id"
		where += " AND collection_items.collection_id = ?"
		args = append(args, query.CollectionID)
	}
	if query.LibraryID != "" {
		where += " AND media_items.library_id = ?"
		args = append(args, query.LibraryID)
	}
	if query.MediaType != "" {
		where += " AND media_items.media_type = ?"
		args = append(args, query.MediaType)
	}
	if query.Title != "" {
		where += " AND LOWER(COALESCE(media_items.title, '') || ' ' || COALESCE(media_items.display_title, '') || ' ' || COALESCE(media_items.original_title, '')) LIKE ?"
		args = append(args, "%"+strings.ToLower(query.Title)+"%")
	}
	if query.Year != 0 {
		where += " AND media_items.year = ?"
		args = append(args, query.Year)
	}
	if query.MatchStatus != "" {
		where += " AND media_items.match_status = ?"
		args = append(args, string(query.MatchStatus))
	}
	return itemSelect() + joins + "\n" + where + "\nGROUP BY media_items.id\nORDER BY media_items.updated_at DESC, media_items.id ASC", args
}

func scanItem(scanner scanner) (Item, error) {
	var item Item
	var title, originalTitle, displayTitle, overview, originalLang, releaseDate sql.NullString
	var year, runtime, versionCount, availableFiles, missingFiles sql.NullInt64
	var matchStatus string
	var locked int
	var createdAt, updatedAt string
	if err := scanner.Scan(
		&item.ID, &item.LibraryID, &item.MediaType, &title, &originalTitle, &displayTitle,
		&year, &overview, &originalLang, &item.DisplayLang, &releaseDate, &runtime,
		&item.Status, &matchStatus, &locked, &createdAt, &updatedAt,
		&versionCount, &availableFiles, &missingFiles,
	); err != nil {
		return Item{}, err
	}
	item.Title = title.String
	item.OriginalTitle = originalTitle.String
	item.DisplayTitle = displayTitle.String
	item.Year = int(year.Int64)
	item.Overview = overview.String
	item.OriginalLang = originalLang.String
	item.ReleaseDate = releaseDate.String
	item.Runtime = int(runtime.Int64)
	item.MatchStatus = MatchStatus(matchStatus)
	item.Locked = locked == 1
	item.VersionCount = int(versionCount.Int64)
	item.AvailableFiles = int(availableFiles.Int64)
	item.MissingFiles = int(missingFiles.Int64)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	return item, nil
}

func scanVersion(scanner scanner) (Version, error) {
	var version Version
	var name, resolution, source, videoCodec, audioCodec, hdrFormat, edition, releaseGroup sql.NullString
	var audioLanguages, subtitleFlags sql.NullString
	var qualityScore sql.NullInt64
	var isDefault int
	var createdAt, updatedAt string
	if err := scanner.Scan(
		&version.ID, &version.MediaID, &name, &resolution, &source, &videoCodec, &audioCodec,
		&hdrFormat, &edition, &releaseGroup, &audioLanguages, &subtitleFlags, &qualityScore,
		&isDefault, &createdAt, &updatedAt,
	); err != nil {
		return Version{}, err
	}
	version.Name = name.String
	version.Resolution = resolution.String
	version.Source = source.String
	version.VideoCodec = videoCodec.String
	version.AudioCodec = audioCodec.String
	version.HDRFormat = hdrFormat.String
	version.Edition = edition.String
	version.ReleaseGroup = releaseGroup.String
	version.AudioLanguages = audioLanguages.String
	version.SubtitleFlags = subtitleFlags.String
	version.QualityScore = int(qualityScore.Int64)
	version.IsDefault = isDefault == 1
	version.CreatedAt = parseTime(createdAt)
	version.UpdatedAt = parseTime(updatedAt)
	return version, nil
}

func scanPerson(scanner scanner) (Person, error) {
	var person Person
	var originalName, localizedName, gender, avatar, bio, birthDate sql.NullString
	var createdAt, updatedAt string
	if err := scanner.Scan(&person.ID, &person.Name, &originalName, &localizedName, &gender, &avatar, &bio, &birthDate, &createdAt, &updatedAt); err != nil {
		return Person{}, err
	}
	person.OriginalName = originalName.String
	person.LocalizedName = localizedName.String
	person.Gender = gender.String
	person.Avatar = avatar.String
	person.Bio = bio.String
	person.BirthDate = birthDate.String
	person.CreatedAt = parseTime(createdAt)
	person.UpdatedAt = parseTime(updatedAt)
	return person, nil
}

func scanTag(scanner scanner) (Tag, error) {
	var tag Tag
	var category sql.NullString
	var createdAt, updatedAt string
	if err := scanner.Scan(&tag.ID, &tag.Name, &tag.NormalizedName, &category, &createdAt, &updatedAt); err != nil {
		return Tag{}, err
	}
	tag.Category = category.String
	tag.CreatedAt = parseTime(createdAt)
	tag.UpdatedAt = parseTime(updatedAt)
	return tag, nil
}

func scanCollection(scanner scanner) (Collection, error) {
	var collection Collection
	var localizedName, description, poster, source, externalID sql.NullString
	var locked int
	var createdAt, updatedAt string
	if err := scanner.Scan(&collection.ID, &collection.Name, &localizedName, &collection.Type, &description, &poster, &source, &externalID, &locked, &createdAt, &updatedAt); err != nil {
		return Collection{}, err
	}
	collection.LocalizedName = localizedName.String
	collection.Description = description.String
	collection.Poster = poster.String
	collection.Source = source.String
	collection.ExternalID = externalID.String
	collection.Locked = locked == 1
	collection.CreatedAt = parseTime(createdAt)
	collection.UpdatedAt = parseTime(updatedAt)
	return collection, nil
}

func scanExternalID(scanner scanner) (ExternalID, error) {
	var externalID ExternalID
	var url sql.NullString
	var createdAt string
	if err := scanner.Scan(&externalID.ID, &externalID.EntityType, &externalID.EntityID, &externalID.Provider, &externalID.ExternalID, &url, &createdAt); err != nil {
		return ExternalID{}, err
	}
	externalID.URL = url.String
	externalID.CreatedAt = parseTime(createdAt)
	return externalID, nil
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
