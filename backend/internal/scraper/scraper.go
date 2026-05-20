package scraper

import "context"

type SearchQuery struct {
	MediaType string
	Title     string
	Year      int
	Number    string
	Language  string
}

type Candidate struct {
	Provider      string   `json:"provider"`
	ExternalID    string   `json:"external_id"`
	Number        string   `json:"number,omitempty"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Year          int      `json:"year"`
	PosterURL     string   `json:"poster_url"`
	Overview      string   `json:"overview"`
	Score         int      `json:"score"`
	ScoreReasons  []string `json:"score_reasons"`
}

type Metadata struct {
	Provider       string   `json:"provider"`
	ExternalID     string   `json:"external_id"`
	Number         string   `json:"number,omitempty"`
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"original_title"`
	DisplayTitle   string   `json:"display_title"`
	Language       string   `json:"language"`
	Overview       string   `json:"overview"`
	Year           int      `json:"year"`
	PosterURL      string   `json:"poster_url,omitempty"`
	ReleaseDate    string   `json:"release_date,omitempty"`
	RuntimeMinutes int      `json:"runtime_minutes,omitempty"`
	Studio         string   `json:"studio,omitempty"`
	Series         string   `json:"series,omitempty"`
	Actors         []string `json:"actors,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

type Scraper interface {
	Name() string
	Supports(mediaType string) bool
	Search(ctx context.Context, query SearchQuery) ([]Candidate, error)
	Fetch(ctx context.Context, candidate Candidate) (*Metadata, error)
}
