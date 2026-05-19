package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store interface {
	UpsertFile(context.Context, FileInput) (File, error)
	GetFile(context.Context, string) (File, bool, error)
	UpdateFilePath(context.Context, string, string) (File, bool, error)
	ListFilesByLibrary(context.Context, string) ([]File, error)
	ListFiles(context.Context, FileQuery) ([]File, error)
	GetFileByPath(context.Context, string) (File, bool, error)
	MarkMissingByLibrary(context.Context, string, []string) (int, error)
	DeleteMissingByLibrary(context.Context, string) (int, error)
}

type FileQuery struct {
	LibraryID string
	MediaID   string
	Status    FileStatus
}

type FileInput struct {
	MediaID           string
	VersionID         string
	LibraryID         string
	Path              string
	FileName          string
	Extension         string
	Size              int64
	ModifiedAt        time.Time
	DetectedMediaType string
	ParsedTitle       string
	ParsedYear        int
	ParsedSeason      int
	ParsedEpisode     int
	ParsedNumber      string
	IsSTRM            bool
	STRMTarget        string
}

type MemoryStore struct {
	mu    sync.Mutex
	files map[string]File
	now   func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		files: make(map[string]File),
		now:   time.Now,
	}
}

func (s *MemoryStore) UpsertFile(_ context.Context, input FileInput) (File, error) {
	input.Path = strings.TrimSpace(input.Path)
	input.LibraryID = strings.TrimSpace(input.LibraryID)
	if input.LibraryID == "" {
		return File{}, fmt.Errorf("library id is required")
	}
	if input.Path == "" {
		return File{}, fmt.Errorf("file path is required")
	}
	if input.FileName == "" {
		input.FileName = filepath.Base(input.Path)
	}
	if input.Extension == "" {
		input.Extension = strings.ToLower(filepath.Ext(input.Path))
	}

	normalizedPath := normalizePath(input.Path)
	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	file, exists := s.files[normalizedPath]
	if !exists {
		file = File{
			ID:             fallbackID("file"),
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
	file.UpdatedAt = now

	s.files[normalizedPath] = file
	return file, nil
}

func (s *MemoryStore) ListFilesByLibrary(_ context.Context, libraryID string) ([]File, error) {
	return s.ListFiles(context.Background(), FileQuery{LibraryID: libraryID})
}

func (s *MemoryStore) ListFiles(_ context.Context, query FileQuery) ([]File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files := make([]File, 0)
	for _, file := range s.files {
		if query.LibraryID != "" && file.LibraryID != query.LibraryID {
			continue
		}
		if query.MediaID != "" && file.MediaID != query.MediaID {
			continue
		}
		if query.Status != "" && file.Status != query.Status {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

func (s *MemoryStore) GetFile(_ context.Context, id string) (File, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, file := range s.files {
		if file.ID == id {
			return file, true, nil
		}
	}
	return File{}, false, nil
}

func (s *MemoryStore) UpdateFilePath(_ context.Context, id string, path string) (File, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return File{}, false, fmt.Errorf("file path is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	for key, file := range s.files {
		if file.ID != id {
			continue
		}
		delete(s.files, key)
		file.Path = path
		file.NormalizedPath = normalizePath(path)
		file.FileName = filepath.Base(path)
		file.Extension = strings.ToLower(filepath.Ext(path))
		file.Status = FileStatusAvailable
		file.UpdatedAt = now
		s.files[file.NormalizedPath] = file
		return file, true, nil
	}
	return File{}, false, nil
}

func (s *MemoryStore) GetFileByPath(_ context.Context, path string) (File, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, ok := s.files[normalizePath(path)]
	return file, ok, nil
}

func (s *MemoryStore) MarkMissingByLibrary(_ context.Context, libraryID string, availablePaths []string) (int, error) {
	available := make(map[string]struct{}, len(availablePaths))
	for _, path := range availablePaths {
		available[normalizePath(path)] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	changed := 0
	for key, file := range s.files {
		if file.LibraryID != libraryID {
			continue
		}
		if _, ok := available[key]; ok {
			continue
		}
		if file.Status == FileStatusMissing {
			continue
		}
		file.Status = FileStatusMissing
		file.UpdatedAt = now
		s.files[key] = file
		changed++
	}
	return changed, nil
}

func (s *MemoryStore) DeleteMissingByLibrary(_ context.Context, libraryID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0
	for key, file := range s.files {
		if file.LibraryID == libraryID && file.Status == FileStatusMissing {
			delete(s.files, key)
			deleted++
		}
	}
	return deleted, nil
}

func normalizePath(path string) string {
	return filepath.Clean(path)
}

func fallbackID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
