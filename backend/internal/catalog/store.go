package catalog

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store interface {
	CreateItem(context.Context, ItemInput) (Item, error)
	GetItem(context.Context, string) (Item, bool, error)
	ListItems(context.Context, ItemQuery) ([]Item, error)
	UpdateItem(context.Context, string, ItemUpdate) (Item, bool, error)
	CreateVersion(context.Context, string, VersionInput) (Version, error)
	ListVersions(context.Context, string) ([]Version, error)
	SetDefaultVersion(context.Context, string, string) (Version, bool, error)
	CreatePerson(context.Context, PersonInput) (Person, error)
	GetPerson(context.Context, string) (Person, bool, error)
	ListPeople(context.Context) ([]Person, error)
	AddMediaPerson(context.Context, string, MediaPersonInput) error
	CreateTag(context.Context, TagInput) (Tag, error)
	GetTag(context.Context, string) (Tag, bool, error)
	ListTags(context.Context) ([]Tag, error)
	AddMediaTag(context.Context, string, MediaTagInput) error
	CreateCollection(context.Context, CollectionInput) (Collection, error)
	GetCollection(context.Context, string) (Collection, bool, error)
	ListCollections(context.Context) ([]Collection, error)
	AddCollectionItem(context.Context, string, CollectionItemInput) error
	UpsertExternalID(context.Context, ExternalIDInput) (ExternalID, error)
	ListExternalIDs(context.Context, string, string) ([]ExternalID, error)
}

type ItemQuery struct {
	LibraryID    string
	MediaType    string
	Title        string
	Year         int
	MatchStatus  MatchStatus
	PersonID     string
	TagID        string
	CollectionID string
}

type ItemInput struct {
	LibraryID     string      `json:"library_id"`
	MediaType     string      `json:"media_type"`
	Title         string      `json:"title"`
	OriginalTitle string      `json:"original_title"`
	DisplayTitle  string      `json:"display_title"`
	Year          int         `json:"year"`
	Overview      string      `json:"overview"`
	OriginalLang  string      `json:"original_language"`
	DisplayLang   string      `json:"display_language"`
	ReleaseDate   string      `json:"release_date"`
	Runtime       int         `json:"runtime"`
	Status        string      `json:"status"`
	MatchStatus   MatchStatus `json:"match_status"`
	Locked        bool        `json:"locked"`
}

type ItemUpdate struct {
	Title         *string      `json:"title"`
	OriginalTitle *string      `json:"original_title"`
	DisplayTitle  *string      `json:"display_title"`
	Year          *int         `json:"year"`
	Overview      *string      `json:"overview"`
	OriginalLang  *string      `json:"original_language"`
	DisplayLang   *string      `json:"display_language"`
	ReleaseDate   *string      `json:"release_date"`
	Runtime       *int         `json:"runtime"`
	Status        *string      `json:"status"`
	MatchStatus   *MatchStatus `json:"match_status"`
	Locked        *bool        `json:"locked"`
}

type VersionInput struct {
	Name           string `json:"name"`
	Resolution     string `json:"resolution"`
	Source         string `json:"source"`
	VideoCodec     string `json:"video_codec"`
	AudioCodec     string `json:"audio_codec"`
	HDRFormat      string `json:"hdr_format"`
	Edition        string `json:"edition"`
	ReleaseGroup   string `json:"release_group"`
	AudioLanguages string `json:"audio_languages"`
	SubtitleFlags  string `json:"subtitle_flags"`
	QualityScore   int    `json:"quality_score"`
	IsDefault      bool   `json:"is_default"`
}

type MemoryStore struct {
	mu              sync.Mutex
	items           map[string]Item
	versions        map[string]Version
	people          map[string]Person
	tags            map[string]Tag
	sets            map[string]Collection
	mediaPeople     map[string]map[string]bool
	mediaTags       map[string]map[string]bool
	collectionItems map[string]map[string]bool
	externalIDs     map[string]ExternalID
	now             func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items:           make(map[string]Item),
		versions:        make(map[string]Version),
		people:          make(map[string]Person),
		tags:            make(map[string]Tag),
		sets:            make(map[string]Collection),
		mediaPeople:     make(map[string]map[string]bool),
		mediaTags:       make(map[string]map[string]bool),
		collectionItems: make(map[string]map[string]bool),
		externalIDs:     make(map[string]ExternalID),
		now:             time.Now,
	}
}

func (s *MemoryStore) CreateItem(_ context.Context, input ItemInput) (Item, error) {
	item, err := itemFromInput(input, s.now().UTC())
	if err != nil {
		return Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = fmt.Sprintf("media_%d", item.CreatedAt.UnixNano())
	s.items[item.ID] = item
	return item, nil
}

func (s *MemoryStore) GetItem(_ context.Context, id string) (Item, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if ok {
		item = s.withCounts(item)
	}
	return item, ok, nil
}

func (s *MemoryStore) ListItems(_ context.Context, query ItemQuery) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if !matchesItemQuery(item, query) {
			continue
		}
		if query.PersonID != "" && !s.mediaPeople[item.ID][query.PersonID] {
			continue
		}
		if query.TagID != "" && !s.mediaTags[item.ID][query.TagID] {
			continue
		}
		if query.CollectionID != "" && !s.collectionItems[query.CollectionID][item.ID] {
			continue
		}
		items = append(items, s.withCounts(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (s *MemoryStore) UpdateItem(_ context.Context, id string, input ItemUpdate) (Item, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, false, nil
	}
	applyItemUpdate(&item, input)
	item.UpdatedAt = s.now().UTC()
	s.items[id] = item
	return s.withCounts(item), true, nil
}

func (s *MemoryStore) CreateVersion(_ context.Context, mediaID string, input VersionInput) (Version, error) {
	if mediaID == "" {
		return Version{}, fmt.Errorf("media id is required")
	}
	now := s.now().UTC()
	version := Version{
		ID:             fmt.Sprintf("version_%d", now.UnixNano()),
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[mediaID]; !ok {
		return Version{}, fmt.Errorf("media item not found")
	}
	if version.IsDefault {
		s.clearDefaultVersion(mediaID)
	}
	s.versions[version.ID] = version
	return version, nil
}

func (s *MemoryStore) ListVersions(_ context.Context, mediaID string) ([]Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listVersionsLocked(mediaID), nil
}

func (s *MemoryStore) SetDefaultVersion(_ context.Context, mediaID, versionID string) (Version, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	version, ok := s.versions[versionID]
	if !ok || version.MediaID != mediaID {
		return Version{}, false, nil
	}
	s.clearDefaultVersion(mediaID)
	version.IsDefault = true
	version.UpdatedAt = s.now().UTC()
	s.versions[version.ID] = version
	return version, true, nil
}

func (s *MemoryStore) CreatePerson(_ context.Context, input PersonInput) (Person, error) {
	if input.Name == "" {
		return Person{}, fmt.Errorf("person name is required")
	}
	now := s.now().UTC()
	person := Person{ID: fmt.Sprintf("person_%d", now.UnixNano()), Name: input.Name, OriginalName: input.OriginalName, LocalizedName: input.LocalizedName, Gender: input.Gender, Avatar: input.Avatar, Bio: input.Bio, BirthDate: input.BirthDate, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.people[person.ID] = person
	return person, nil
}

func (s *MemoryStore) GetPerson(_ context.Context, id string) (Person, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	person, ok := s.people[id]
	return person, ok, nil
}

func (s *MemoryStore) ListPeople(context.Context) ([]Person, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	people := make([]Person, 0, len(s.people))
	for _, person := range s.people {
		people = append(people, person)
	}
	sort.Slice(people, func(i, j int) bool { return people[i].Name < people[j].Name })
	return people, nil
}

func (s *MemoryStore) AddMediaPerson(_ context.Context, mediaID string, input MediaPersonInput) error {
	if mediaID == "" || input.PersonID == "" {
		return fmt.Errorf("media id and person id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[mediaID]; !ok {
		return fmt.Errorf("media item not found")
	}
	if _, ok := s.people[input.PersonID]; !ok {
		return fmt.Errorf("person not found")
	}
	if s.mediaPeople[mediaID] == nil {
		s.mediaPeople[mediaID] = make(map[string]bool)
	}
	s.mediaPeople[mediaID][input.PersonID] = true
	return nil
}

func (s *MemoryStore) CreateTag(_ context.Context, input TagInput) (Tag, error) {
	if input.Name == "" {
		return Tag{}, fmt.Errorf("tag name is required")
	}
	if input.NormalizedName == "" {
		input.NormalizedName = normalizeName(input.Name)
	}
	now := s.now().UTC()
	tag := Tag{ID: fmt.Sprintf("tag_%d", now.UnixNano()), Name: input.Name, NormalizedName: input.NormalizedName, Category: input.Category, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags[tag.ID] = tag
	return tag, nil
}

func (s *MemoryStore) GetTag(_ context.Context, id string) (Tag, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tag, ok := s.tags[id]
	return tag, ok, nil
}

func (s *MemoryStore) ListTags(context.Context) ([]Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags := make([]Tag, 0, len(s.tags))
	for _, tag := range s.tags {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	return tags, nil
}

func (s *MemoryStore) AddMediaTag(_ context.Context, mediaID string, input MediaTagInput) error {
	if mediaID == "" || input.TagID == "" {
		return fmt.Errorf("media id and tag id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[mediaID]; !ok {
		return fmt.Errorf("media item not found")
	}
	if _, ok := s.tags[input.TagID]; !ok {
		return fmt.Errorf("tag not found")
	}
	if s.mediaTags[mediaID] == nil {
		s.mediaTags[mediaID] = make(map[string]bool)
	}
	s.mediaTags[mediaID][input.TagID] = true
	return nil
}

func (s *MemoryStore) CreateCollection(_ context.Context, input CollectionInput) (Collection, error) {
	if input.Name == "" {
		return Collection{}, fmt.Errorf("collection name is required")
	}
	if input.Type == "" {
		input.Type = "manual"
	}
	now := s.now().UTC()
	collection := Collection{ID: fmt.Sprintf("collection_%d", now.UnixNano()), Name: input.Name, LocalizedName: input.LocalizedName, Type: input.Type, Description: input.Description, Poster: input.Poster, Source: input.Source, ExternalID: input.ExternalID, Locked: input.Locked, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets[collection.ID] = collection
	return collection, nil
}

func (s *MemoryStore) GetCollection(_ context.Context, id string) (Collection, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	collection, ok := s.sets[id]
	return collection, ok, nil
}

func (s *MemoryStore) ListCollections(context.Context) ([]Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	collections := make([]Collection, 0, len(s.sets))
	for _, collection := range s.sets {
		collections = append(collections, collection)
	}
	sort.Slice(collections, func(i, j int) bool { return collections[i].Name < collections[j].Name })
	return collections, nil
}

func (s *MemoryStore) AddCollectionItem(_ context.Context, collectionID string, input CollectionItemInput) error {
	if collectionID == "" || input.MediaID == "" {
		return fmt.Errorf("collection id and media id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sets[collectionID]; !ok {
		return fmt.Errorf("collection not found")
	}
	if _, ok := s.items[input.MediaID]; !ok {
		return fmt.Errorf("media item not found")
	}
	if s.collectionItems[collectionID] == nil {
		s.collectionItems[collectionID] = make(map[string]bool)
	}
	s.collectionItems[collectionID][input.MediaID] = true
	return nil
}

func (s *MemoryStore) UpsertExternalID(_ context.Context, input ExternalIDInput) (ExternalID, error) {
	externalID, err := externalIDFromInput(input, s.now().UTC())
	if err != nil {
		return ExternalID{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := externalIDKey(input.EntityType, input.EntityID, input.Provider)
	if existing, ok := s.externalIDs[key]; ok {
		externalID.ID = existing.ID
		externalID.CreatedAt = existing.CreatedAt
	} else {
		externalID.ID = fmt.Sprintf("external_id_%d", externalID.CreatedAt.UnixNano())
	}
	s.externalIDs[key] = externalID
	return externalID, nil
}

func (s *MemoryStore) ListExternalIDs(_ context.Context, entityType, entityID string) ([]ExternalID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	externalIDs := make([]ExternalID, 0)
	for _, externalID := range s.externalIDs {
		if externalID.EntityType == entityType && externalID.EntityID == entityID {
			externalIDs = append(externalIDs, externalID)
		}
	}
	sort.Slice(externalIDs, func(i, j int) bool {
		if externalIDs[i].Provider == externalIDs[j].Provider {
			return externalIDs[i].ExternalID < externalIDs[j].ExternalID
		}
		return externalIDs[i].Provider < externalIDs[j].Provider
	})
	return externalIDs, nil
}

func itemFromInput(input ItemInput, now time.Time) (Item, error) {
	if input.LibraryID == "" {
		return Item{}, fmt.Errorf("library id is required")
	}
	if input.MediaType == "" {
		return Item{}, fmt.Errorf("media type is required")
	}
	if input.DisplayLang == "" {
		input.DisplayLang = "zh-CN"
	}
	if input.Status == "" {
		input.Status = "pending"
	}
	if input.MatchStatus == "" {
		input.MatchStatus = MatchStatusUnmatched
	}
	return Item{
		LibraryID:     input.LibraryID,
		MediaType:     input.MediaType,
		Title:         input.Title,
		OriginalTitle: input.OriginalTitle,
		DisplayTitle:  input.DisplayTitle,
		Year:          input.Year,
		Overview:      input.Overview,
		OriginalLang:  input.OriginalLang,
		DisplayLang:   input.DisplayLang,
		ReleaseDate:   input.ReleaseDate,
		Runtime:       input.Runtime,
		Status:        input.Status,
		MatchStatus:   input.MatchStatus,
		Locked:        input.Locked,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func externalIDFromInput(input ExternalIDInput, now time.Time) (ExternalID, error) {
	if input.EntityType == "" || input.EntityID == "" {
		return ExternalID{}, fmt.Errorf("entity type and entity id are required")
	}
	if input.Provider == "" || input.ExternalID == "" {
		return ExternalID{}, fmt.Errorf("provider and external id are required")
	}
	return ExternalID{
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		Provider:   input.Provider,
		ExternalID: input.ExternalID,
		URL:        input.URL,
		CreatedAt:  now,
	}, nil
}

func externalIDKey(entityType, entityID, provider string) string {
	return entityType + "\x00" + entityID + "\x00" + provider
}

func applyItemUpdate(item *Item, input ItemUpdate) {
	if input.Title != nil {
		item.Title = *input.Title
	}
	if input.OriginalTitle != nil {
		item.OriginalTitle = *input.OriginalTitle
	}
	if input.DisplayTitle != nil {
		item.DisplayTitle = *input.DisplayTitle
	}
	if input.Year != nil {
		item.Year = *input.Year
	}
	if input.Overview != nil {
		item.Overview = *input.Overview
	}
	if input.OriginalLang != nil {
		item.OriginalLang = *input.OriginalLang
	}
	if input.DisplayLang != nil {
		item.DisplayLang = *input.DisplayLang
	}
	if input.ReleaseDate != nil {
		item.ReleaseDate = *input.ReleaseDate
	}
	if input.Runtime != nil {
		item.Runtime = *input.Runtime
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	if input.MatchStatus != nil {
		item.MatchStatus = *input.MatchStatus
	}
	if input.Locked != nil {
		item.Locked = *input.Locked
	}
}

func matchesItemQuery(item Item, query ItemQuery) bool {
	if query.LibraryID != "" && item.LibraryID != query.LibraryID {
		return false
	}
	if query.MediaType != "" && item.MediaType != query.MediaType {
		return false
	}
	if query.Title != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.DisplayTitle+" "+item.OriginalTitle), strings.ToLower(query.Title)) {
		return false
	}
	if query.Year != 0 && item.Year != query.Year {
		return false
	}
	if query.MatchStatus != "" && item.MatchStatus != query.MatchStatus {
		return false
	}
	return true
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *MemoryStore) withCounts(item Item) Item {
	versions := s.listVersionsLocked(item.ID)
	item.VersionCount = len(versions)
	for _, version := range versions {
		if version.IsDefault {
			v := version
			item.DefaultVersion = &v
			break
		}
	}
	return item
}

func (s *MemoryStore) listVersionsLocked(mediaID string) []Version {
	versions := make([]Version, 0)
	for _, version := range s.versions {
		if version.MediaID == mediaID {
			versions = append(versions, version)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].IsDefault != versions[j].IsDefault {
			return versions[i].IsDefault
		}
		if versions[i].QualityScore != versions[j].QualityScore {
			return versions[i].QualityScore > versions[j].QualityScore
		}
		return versions[i].CreatedAt.Before(versions[j].CreatedAt)
	})
	return versions
}

func (s *MemoryStore) clearDefaultVersion(mediaID string) {
	for id, version := range s.versions {
		if version.MediaID == mediaID {
			version.IsDefault = false
			version.UpdatedAt = s.now().UTC()
			s.versions[id] = version
		}
	}
}
