package download

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Create(context.Context, DirectoryInput) (Directory, error)
	Get(context.Context, string) (Directory, bool, error)
	List(context.Context) ([]Directory, error)
	ListWatchEnabled(context.Context) ([]Directory, error)
	Update(context.Context, string, DirectoryUpdate) (Directory, bool, error)
	Delete(context.Context, string) (bool, error)
}

type MemoryStore struct {
	mu          sync.Mutex
	directories map[string]Directory
	now         func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		directories: make(map[string]Directory),
		now:         time.Now,
	}
}

func (s *MemoryStore) Create(_ context.Context, input DirectoryInput) (Directory, error) {
	directory, err := directoryFromInput(input, s.now().UTC())
	if err != nil {
		return Directory{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	directory.ID = fmt.Sprintf("download_dir_%d", directory.CreatedAt.UnixNano())
	s.directories[directory.ID] = directory
	return directory, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Directory, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directory, ok := s.directories[id]
	return directory, ok, nil
}

func (s *MemoryStore) List(context.Context) ([]Directory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directories := make([]Directory, 0, len(s.directories))
	for _, directory := range s.directories {
		directories = append(directories, directory)
	}
	sortDirectories(directories)
	return directories, nil
}

func (s *MemoryStore) ListWatchEnabled(context.Context) ([]Directory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directories := make([]Directory, 0, len(s.directories))
	for _, directory := range s.directories {
		if directory.Enabled && directory.WatchEnabled {
			directories = append(directories, directory)
		}
	}
	sortDirectories(directories)
	return directories, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input DirectoryUpdate) (Directory, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	directory, ok := s.directories[id]
	if !ok {
		return Directory{}, false, nil
	}
	applyUpdate(&directory, input)
	if err := validate(directory); err != nil {
		return Directory{}, true, err
	}
	directory.UpdatedAt = s.now().UTC()
	s.directories[id] = directory
	return directory, true, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.directories[id]; !ok {
		return false, nil
	}
	delete(s.directories, id)
	return true, nil
}

func directoryFromInput(input DirectoryInput, now time.Time) (Directory, error) {
	directory := Directory{
		Name:         strings.TrimSpace(input.Name),
		Path:         strings.TrimSpace(input.Path),
		LibraryID:    strings.TrimSpace(input.LibraryID),
		MediaType:    strings.TrimSpace(input.MediaType),
		ActionMode:   defaultString(strings.TrimSpace(input.ActionMode), "hardlink"),
		Enabled:      input.Enabled,
		WatchEnabled: input.WatchEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := validate(directory); err != nil {
		return Directory{}, err
	}
	return directory, nil
}

func applyUpdate(directory *Directory, input DirectoryUpdate) {
	if input.Name != nil {
		directory.Name = strings.TrimSpace(*input.Name)
	}
	if input.Path != nil {
		directory.Path = strings.TrimSpace(*input.Path)
	}
	if input.LibraryID != nil {
		directory.LibraryID = strings.TrimSpace(*input.LibraryID)
	}
	if input.MediaType != nil {
		directory.MediaType = strings.TrimSpace(*input.MediaType)
	}
	if input.ActionMode != nil {
		directory.ActionMode = strings.TrimSpace(*input.ActionMode)
	}
	if input.Enabled != nil {
		directory.Enabled = *input.Enabled
	}
	if input.WatchEnabled != nil {
		directory.WatchEnabled = *input.WatchEnabled
	}
}

func validate(directory Directory) error {
	if directory.Name == "" {
		return fmt.Errorf("download directory name is required")
	}
	if directory.Path == "" {
		return fmt.Errorf("download directory path is required")
	}
	if directory.LibraryID == "" {
		return fmt.Errorf("target library id is required")
	}
	if directory.ActionMode == "" {
		return fmt.Errorf("action mode is required")
	}
	switch directory.ActionMode {
	case "move", "copy", "hardlink", "symlink":
	default:
		return fmt.Errorf("unsupported action mode %q", directory.ActionMode)
	}
	return nil
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func sortDirectories(directories []Directory) {
	sort.Slice(directories, func(i, j int) bool {
		if directories[i].CreatedAt.Equal(directories[j].CreatedAt) {
			return directories[i].ID < directories[j].ID
		}
		return directories[i].CreatedAt.Before(directories[j].CreatedAt)
	})
}
