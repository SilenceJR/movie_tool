package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

var mediaExtensions = map[string]struct{}{
	".3gp":  {},
	".avi":  {},
	".flv":  {},
	".iso":  {},
	".m2ts": {},
	".m4v":  {},
	".mkv":  {},
	".mov":  {},
	".mp4":  {},
	".mpeg": {},
	".mpg":  {},
	".rmvb": {},
	".ts":   {},
	".webm": {},
	".wmv":  {},
}

type LibraryInfo struct {
	ID        string
	Name      string
	Path      string
	MediaType string
}

type ScanRequest struct {
	Root           string
	Library        LibraryInfo
	MinModifiedAge time.Duration
	Now            func() time.Time
}

type Scanner struct{}

func NewScanner() Scanner {
	return Scanner{}
}

func Walk(root string, library LibraryInfo) ([]ParsedFile, error) {
	return NewScanner().Walk(ScanRequest{
		Root:    root,
		Library: library,
	})
}

func (s Scanner) Walk(request ScanRequest) ([]ParsedFile, error) {
	root := request.Root
	if root == "" {
		root = request.Library.Path
	}
	if root == "" {
		return nil, fmt.Errorf("scan root is required")
	}
	now := time.Now
	if request.Now != nil {
		now = request.Now
	}
	stableSince := time.Time{}
	if request.MinModifiedAge > 0 {
		stableSince = now().Add(-request.MinModifiedAge)
	}

	var files []ParsedFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if isHidden(path, root, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !isMediaFile(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !stableSince.IsZero() && info.ModTime().After(stableSince) {
			return nil
		}

		parsed := ParseFile(path)
		parsed.LibraryID = request.Library.ID
		parsed.LibraryName = request.Library.Name
		parsed.MediaType = request.Library.MediaType
		parsed.Size = info.Size()
		parsed.ModifiedAt = info.ModTime()
		files = append(files, parsed)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func isMediaFile(name string) bool {
	_, ok := mediaExtensions[strings.ToLower(filepath.Ext(name))]
	return ok
}

func isHidden(path, root string, entry fs.DirEntry) bool {
	if path == root {
		return false
	}
	return strings.HasPrefix(entry.Name(), ".")
}
