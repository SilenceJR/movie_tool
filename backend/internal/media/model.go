package media

import "time"

type FileStatus string

const (
	FileStatusAvailable FileStatus = "available"
	FileStatusMissing   FileStatus = "missing"
	FileStatusDeleted   FileStatus = "deleted"
	FileStatusIgnored   FileStatus = "ignored"
	FileStatusPending   FileStatus = "pending"
	FileStatusFailed    FileStatus = "failed"
)

type MatchStatus string

const (
	MatchStatusMatched       MatchStatus = "matched"
	MatchStatusAmbiguous     MatchStatus = "ambiguous"
	MatchStatusLowConfidence MatchStatus = "low_confidence"
	MatchStatusUnmatched     MatchStatus = "unmatched"
	MatchStatusLocked        MatchStatus = "locked"
)

type Item struct {
	ID            string
	LibraryID     string
	MediaType     string
	Title         string
	OriginalTitle string
	DisplayTitle  string
	Year          int
	Overview      string
	OriginalLang  string
	DisplayLang   string
	ReleaseDate   string
	Runtime       int
	Status        string
	MatchStatus   MatchStatus
	Locked        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Version struct {
	ID             string
	MediaID        string
	Name           string
	Resolution     string
	Source         string
	VideoCodec     string
	AudioCodec     string
	HDRFormat      string
	Edition        string
	ReleaseGroup   string
	AudioLanguages []string
	SubtitleFlags  []string
	QualityScore   int
	IsDefault      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type File struct {
	ID                string
	MediaID           string
	VersionID         string
	LibraryID         string
	Path              string
	NormalizedPath    string
	FileName          string
	Extension         string
	Size              int64
	ModifiedAt        time.Time
	Status            FileStatus
	IsSTRM            bool
	STRMTarget        string
	DetectedMediaType string
	ParsedTitle       string
	ParsedYear        int
	ParsedSeason      int
	ParsedEpisode     int
	ParsedNumber      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
