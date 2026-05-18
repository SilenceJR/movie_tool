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
	Provider      string
	ExternalID    string
	Title         string
	OriginalTitle string
	Year          int
	PosterURL     string
	Overview      string
	Score         int
	ScoreReasons  []string
}

type Metadata struct {
	Provider      string
	ExternalID    string
	Title         string
	OriginalTitle string
	DisplayTitle  string
	Language      string
	Overview      string
	Year          int
}

type Scraper interface {
	Name() string
	Supports(mediaType string) bool
	Search(ctx context.Context, query SearchQuery) ([]Candidate, error)
	Fetch(ctx context.Context, candidate Candidate) (*Metadata, error)
}
