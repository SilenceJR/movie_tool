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
	ListFilesByLibrary(context.Context, string) ([]File, error)
	GetFileByPath(context.Context, string) (File, bool, error)
}

type FileInput struct {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	files := make([]File, 0)
	for _, file := range s.files {
		if file.LibraryID == libraryID {
			files = append(files, file)
		}
	}
	return files, nil
}

func (s *MemoryStore) GetFileByPath(_ context.Context, path string) (File, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, ok := s.files[normalizePath(path)]
	return file, ok, nil
}

func normalizePath(path string) string {
	return filepath.Clean(path)
}

func fallbackID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
