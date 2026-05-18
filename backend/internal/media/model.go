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
	ID                string     `json:"id"`
	MediaID           string     `json:"media_id,omitempty"`
	VersionID         string     `json:"version_id,omitempty"`
	LibraryID         string     `json:"library_id"`
	Path              string     `json:"path"`
	NormalizedPath    string     `json:"normalized_path"`
	FileName          string     `json:"file_name"`
	Extension         string     `json:"extension"`
	Size              int64      `json:"size"`
	ModifiedAt        time.Time  `json:"modified_at"`
	Status            FileStatus `json:"file_status"`
	IsSTRM            bool       `json:"is_strm"`
	STRMTarget        string     `json:"strm_target,omitempty"`
	DetectedMediaType string     `json:"detected_media_type"`
	ParsedTitle       string     `json:"parsed_title"`
	ParsedYear        int        `json:"parsed_year"`
	ParsedSeason      int        `json:"parsed_season"`
	ParsedEpisode     int        `json:"parsed_episode"`
	ParsedNumber      string     `json:"parsed_number"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
