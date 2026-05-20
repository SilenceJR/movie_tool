package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	TMDBProvider       = "tmdb"
	defaultTMDBBaseURL = "https://api.themoviedb.org"
	defaultPosterBase  = "https://image.tmdb.org/t/p/w500"
)

type TMDBClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func (c TMDBClient) Name() string {
	return TMDBProvider
}

func (c TMDBClient) Supports(mediaType string) bool {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "movie", "tv", "series":
		return true
	default:
		return false
	}
}

func (c TMDBClient) Search(ctx context.Context, query SearchQuery) ([]Candidate, error) {
	mediaType := normalizeTMDBMediaType(query.MediaType)
	if mediaType == "" {
		return nil, fmt.Errorf("tmdb supports movie and tv media types")
	}
	title := strings.TrimSpace(query.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	values := url.Values{}
	values.Set("query", title)
	if query.Language != "" {
		values.Set("language", query.Language)
	}
	if query.Year > 0 {
		if mediaType == "tv" {
			values.Set("first_air_date_year", strconv.Itoa(query.Year))
		} else {
			values.Set("year", strconv.Itoa(query.Year))
		}
	}

	var response tmdbSearchResponse
	if err := c.get(ctx, "/3/search/"+mediaType, values, &response); err != nil {
		return nil, err
	}

	candidates := make([]Candidate, 0, len(response.Results))
	for _, result := range response.Results {
		candidates = append(candidates, result.toCandidate(mediaType))
	}
	return candidates, nil
}

func (c TMDBClient) Fetch(ctx context.Context, candidate Candidate) (*Metadata, error) {
	mediaType := "movie"
	externalID := strings.TrimSpace(candidate.ExternalID)
	if strings.HasPrefix(strings.ToLower(externalID), "tv:") {
		mediaType = "tv"
		externalID = externalID[3:]
	}
	return c.FetchByID(ctx, mediaType, externalID, "")
}

func (c TMDBClient) FetchByID(ctx context.Context, mediaType string, externalID string, language string) (*Metadata, error) {
	mediaType = normalizeTMDBMediaType(mediaType)
	if mediaType == "" {
		return nil, fmt.Errorf("tmdb supports movie and tv media types")
	}
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, fmt.Errorf("external_id is required")
	}

	values := url.Values{}
	if language != "" {
		values.Set("language", language)
	}
	var response tmdbDetailResponse
	if err := c.get(ctx, "/3/"+mediaType+"/"+url.PathEscape(externalID), values, &response); err != nil {
		return nil, err
	}
	return response.toMetadata(mediaType), nil
}

func (c TMDBClient) get(ctx context.Context, path string, values url.Values, target any) error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("tmdb api key is required")
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultTMDBBaseURL
	}
	endpoint, err := url.Parse(baseURL + path)
	if err != nil {
		return err
	}
	endpoint.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("tmdb request failed: status=%d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

type tmdbSearchResponse struct {
	Results []tmdbResult `json:"results"`
}

type tmdbResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
}

func (r tmdbResult) toCandidate(mediaType string) Candidate {
	externalID := strconv.Itoa(r.ID)
	if mediaType == "tv" {
		externalID = "tv:" + externalID
	}
	return Candidate{
		Provider:      TMDBProvider,
		ExternalID:    externalID,
		Title:         firstNonEmptyString(r.Title, r.Name),
		OriginalTitle: firstNonEmptyString(r.OriginalTitle, r.OriginalName),
		Year:          yearFromDate(firstNonEmptyString(r.ReleaseDate, r.FirstAirDate)),
		PosterURL:     tmdbPosterURL(r.PosterPath),
		Overview:      r.Overview,
	}
}

type tmdbDetailResponse struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	Overview      string `json:"overview"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
}

func (r tmdbDetailResponse) toMetadata(mediaType string) *Metadata {
	externalID := strconv.Itoa(r.ID)
	if mediaType == "tv" {
		externalID = "tv:" + externalID
	}
	title := firstNonEmptyString(r.Title, r.Name)
	return &Metadata{
		Provider:      TMDBProvider,
		ExternalID:    externalID,
		Title:         title,
		OriginalTitle: firstNonEmptyString(r.OriginalTitle, r.OriginalName),
		DisplayTitle:  title,
		Overview:      r.Overview,
		Year:          yearFromDate(firstNonEmptyString(r.ReleaseDate, r.FirstAirDate)),
	}
}

func normalizeTMDBMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "movie":
		return "movie"
	case "tv", "series":
		return "tv"
	default:
		return ""
	}
}

func tmdbPosterURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return defaultPosterBase + path
}

func yearFromDate(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return 0
	}
	return year
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
