package library

import "time"

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
	MediaTypeAV    MediaType = "av"
	MediaTypeOther MediaType = "other"
)

type Library struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	MediaType         MediaType `json:"media_type"`
	Path              string    `json:"path"`
	Language          string    `json:"language"`
	FallbackLanguages []string  `json:"fallback_languages,omitempty"`
	CachePolicy       string    `json:"cache_policy"`
	NFOEnabled        bool      `json:"nfo_enabled"`
	STRMEnabled       bool      `json:"strm_enabled"`
	WatchEnabled      bool      `json:"watch_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
