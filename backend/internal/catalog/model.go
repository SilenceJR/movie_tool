package catalog

import "time"

type MatchStatus string

const (
	MatchStatusMatched       MatchStatus = "matched"
	MatchStatusAmbiguous     MatchStatus = "ambiguous"
	MatchStatusLowConfidence MatchStatus = "low_confidence"
	MatchStatusUnmatched     MatchStatus = "unmatched"
	MatchStatusLocked        MatchStatus = "locked"
)

type Item struct {
	ID             string      `json:"id"`
	LibraryID      string      `json:"library_id"`
	MediaType      string      `json:"media_type"`
	Title          string      `json:"title,omitempty"`
	OriginalTitle  string      `json:"original_title,omitempty"`
	DisplayTitle   string      `json:"display_title,omitempty"`
	Year           int         `json:"year,omitempty"`
	Overview       string      `json:"overview,omitempty"`
	OriginalLang   string      `json:"original_language,omitempty"`
	DisplayLang    string      `json:"display_language"`
	ReleaseDate    string      `json:"release_date,omitempty"`
	Runtime        int         `json:"runtime,omitempty"`
	Status         string      `json:"status"`
	MatchStatus    MatchStatus `json:"match_status"`
	Locked         bool        `json:"locked"`
	DefaultVersion *Version    `json:"default_version,omitempty"`
	VersionCount   int         `json:"version_count"`
	AvailableFiles int         `json:"available_files"`
	MissingFiles   int         `json:"missing_files"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type Version struct {
	ID             string    `json:"id"`
	MediaID        string    `json:"media_id"`
	Name           string    `json:"name,omitempty"`
	Resolution     string    `json:"resolution,omitempty"`
	Source         string    `json:"source,omitempty"`
	VideoCodec     string    `json:"video_codec,omitempty"`
	AudioCodec     string    `json:"audio_codec,omitempty"`
	HDRFormat      string    `json:"hdr_format,omitempty"`
	Edition        string    `json:"edition,omitempty"`
	ReleaseGroup   string    `json:"release_group,omitempty"`
	AudioLanguages string    `json:"audio_languages,omitempty"`
	SubtitleFlags  string    `json:"subtitle_flags,omitempty"`
	QualityScore   int       `json:"quality_score,omitempty"`
	IsDefault      bool      `json:"is_default"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Person struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	OriginalName  string    `json:"original_name,omitempty"`
	LocalizedName string    `json:"localized_name,omitempty"`
	Gender        string    `json:"gender,omitempty"`
	Avatar        string    `json:"avatar,omitempty"`
	Bio           string    `json:"bio,omitempty"`
	BirthDate     string    `json:"birth_date,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Tag struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	NormalizedName string    `json:"normalized_name"`
	Category       string    `json:"category,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Collection struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	LocalizedName string    `json:"localized_name,omitempty"`
	Type          string    `json:"type"`
	Description   string    `json:"description,omitempty"`
	Poster        string    `json:"poster,omitempty"`
	Source        string    `json:"source,omitempty"`
	ExternalID    string    `json:"external_id,omitempty"`
	Locked        bool      `json:"locked"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PersonInput struct {
	Name          string `json:"name"`
	OriginalName  string `json:"original_name"`
	LocalizedName string `json:"localized_name"`
	Gender        string `json:"gender"`
	Avatar        string `json:"avatar"`
	Bio           string `json:"bio"`
	BirthDate     string `json:"birth_date"`
}

type TagInput struct {
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	Category       string `json:"category"`
}

type CollectionInput struct {
	Name          string `json:"name"`
	LocalizedName string `json:"localized_name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Poster        string `json:"poster"`
	Source        string `json:"source"`
	ExternalID    string `json:"external_id"`
	Locked        bool   `json:"locked"`
}

type MediaPersonInput struct {
	PersonID      string `json:"person_id"`
	Role          string `json:"role"`
	CharacterName string `json:"character_name"`
	SortOrder     int    `json:"sort_order"`
}

type MediaTagInput struct {
	TagID  string `json:"tag_id"`
	Source string `json:"source"`
}

type CollectionItemInput struct {
	MediaID      string `json:"media_id"`
	SortOrder    int    `json:"sort_order"`
	RelationType string `json:"relation_type"`
}
