package library

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Store interface {
	List(context.Context) ([]Library, error)
	Get(context.Context, string) (Library, bool, error)
	Create(context.Context, CreateInput) (Library, error)
	Update(context.Context, string, UpdateInput) (Library, bool, error)
	Delete(context.Context, string) (bool, error)
}

type CreateInput struct {
	Name         string    `json:"name"`
	MediaType    MediaType `json:"media_type"`
	Path         string    `json:"path"`
	Language     string    `json:"language"`
	CachePolicy  string    `json:"cache_policy"`
	NFOEnabled   bool      `json:"nfo_enabled"`
	STRMEnabled  bool      `json:"strm_enabled"`
	WatchEnabled bool      `json:"watch_enabled"`
}

type UpdateInput struct {
	Name         *string    `json:"name"`
	MediaType    *MediaType `json:"media_type"`
	Path         *string    `json:"path"`
	Language     *string    `json:"language"`
	CachePolicy  *string    `json:"cache_policy"`
	NFOEnabled   *bool      `json:"nfo_enabled"`
	STRMEnabled  *bool      `json:"strm_enabled"`
	WatchEnabled *bool      `json:"watch_enabled"`
}

type MemoryStore struct {
	mu        sync.Mutex
	libraries map[string]Library
	now       func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		libraries: make(map[string]Library),
		now:       time.Now,
	}
}

func (s *MemoryStore) List(context.Context) ([]Library, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	libraries := make([]Library, 0, len(s.libraries))
	for _, library := range s.libraries {
		libraries = append(libraries, library)
	}
	return libraries, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Library, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	library, ok := s.libraries[id]
	return library, ok, nil
}

func (s *MemoryStore) Create(_ context.Context, input CreateInput) (Library, error) {
	if err := normalizeCreateInput(&input); err != nil {
		return Library{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	library := Library{
		ID:           fmt.Sprintf("%d", now.UnixNano()),
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
	s.libraries[library.ID] = library
	return library, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input UpdateInput) (Library, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	library, ok := s.libraries[id]
	if !ok {
		return Library{}, false, nil
	}
	if input.Name != nil {
		library.Name = *input.Name
	}
	if input.MediaType != nil {
		library.MediaType = *input.MediaType
	}
	if input.Path != nil {
		library.Path = *input.Path
	}
	if input.Language != nil {
		library.Language = *input.Language
	}
	if input.CachePolicy != nil {
		library.CachePolicy = *input.CachePolicy
	}
	if input.NFOEnabled != nil {
		library.NFOEnabled = *input.NFOEnabled
	}
	if input.STRMEnabled != nil {
		library.STRMEnabled = *input.STRMEnabled
	}
	if input.WatchEnabled != nil {
		library.WatchEnabled = *input.WatchEnabled
	}
	library.UpdatedAt = s.now()
	s.libraries[id] = library
	return library, true, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.libraries[id]; !ok {
		return false, nil
	}
	delete(s.libraries, id)
	return true, nil
}
