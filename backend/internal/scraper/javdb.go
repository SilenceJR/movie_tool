package scraper

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	JavDBProvider       = "javdb"
	defaultJavDBBaseURL = "https://javdb.com"
)

type JavDBClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c JavDBClient) Name() string {
	return JavDBProvider
}

func (c JavDBClient) Supports(mediaType string) bool {
	return strings.EqualFold(strings.TrimSpace(mediaType), "av")
}

func (c JavDBClient) Search(ctx context.Context, query SearchQuery) ([]Candidate, error) {
	number, ok := ParseAVNumber(firstNonEmptyString(query.Number, query.Title))
	if !ok {
		return nil, fmt.Errorf("av number could not be parsed")
	}
	values := url.Values{}
	values.Set("q", number.Normalized)
	values.Set("f", "all")
	body, err := c.get(ctx, "/search", values)
	if err != nil {
		return nil, err
	}
	return parseJavDBSearch(body, number), nil
}

func (c JavDBClient) Fetch(ctx context.Context, candidate Candidate) (*Metadata, error) {
	return c.FetchByID(ctx, candidate.ExternalID)
}

func (c JavDBClient) FetchByID(ctx context.Context, externalID string) (*Metadata, error) {
	path := strings.TrimSpace(strings.TrimPrefix(externalID, JavDBProvider+":"))
	if path == "" {
		return nil, fmt.Errorf("external_id is required")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	metadata := parseJavDBDetail(body, externalID)
	if metadata.ExternalID == "" {
		metadata.ExternalID = JavDBProvider + ":" + path
	}
	return metadata, nil
}

func (c JavDBClient) get(ctx context.Context, path string, values url.Values) (string, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultJavDBBaseURL
	}
	endpoint, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}
	if values != nil {
		endpoint.RawQuery = values.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("User-Agent", "movie-tool/0.1")

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("javdb request failed: status=%d", response.StatusCode)
	}
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

var (
	javDBCardPattern       = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*/v/[^"']+)["'][^>]*>(.*?)</a>`)
	javDBTitleClassPattern = regexp.MustCompile(`(?is)<[^>]+class=["'][^"']*(?:video-title|title)[^"']*["'][^>]*>(.*?)</[^>]+>`)
	javDBImagePattern      = regexp.MustCompile(`(?is)<img[^>]+(?:src|data-src)=["']([^"']+)["']`)
	javDBDatePattern       = regexp.MustCompile(`\b(19|20)\d{2}[-/]\d{1,2}[-/]\d{1,2}\b`)
	javDBH1Pattern         = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	javDBH2Pattern         = regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	javDBDescriptionPat    = regexp.MustCompile(`(?is)<[^>]+class=["'][^"']*(?:description|overview|summary)[^"']*["'][^>]*>(.*?)</[^>]+>`)
	javDBNumberPattern     = regexp.MustCompile(`(?i)\b([A-Z]{2,8}-\d{2,6}|FC2-PPV-\d{5,}|HEYZO-\d{3,}|(?:CARIB|1PONDO|10MUSUME)-\d{6}-\d{2,3})\b`)
)

func parseJavDBSearch(body string, number AVNumber) []Candidate {
	matches := javDBCardPattern.FindAllStringSubmatch(body, -1)
	candidates := make([]Candidate, 0, len(matches))
	for _, match := range matches {
		path := normalizeJavDBPath(match[1])
		card := match[2]
		title := firstNonEmptyString(extractHTML(javDBTitleClassPattern, card), stripTags(card))
		if title == "" {
			continue
		}
		candidate := Candidate{
			Provider:      JavDBProvider,
			ExternalID:    JavDBProvider + ":" + path,
			Title:         title,
			OriginalTitle: title,
			Year:          yearFromDate(extractText(javDBDatePattern, card)),
			PosterURL:     absolutizeJavDBURL(extractText(javDBImagePattern, card)),
		}
		if strings.Contains(strings.ToUpper(title), number.Normalized) {
			candidate.Score = 90
			candidate.ScoreReasons = []string{"番号精确匹配"}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func parseJavDBDetail(body string, externalID string) *Metadata {
	title := firstNonEmptyString(extractHTML(javDBH1Pattern, body), extractHTML(javDBH2Pattern, body))
	number := strings.ToUpper(extractText(javDBNumberPattern, body))
	displayTitle := strings.TrimSpace(strings.TrimPrefix(title, number))
	return &Metadata{
		Provider:      JavDBProvider,
		ExternalID:    normalizeJavDBExternalID(externalID),
		Title:         firstNonEmptyString(number, title),
		OriginalTitle: title,
		DisplayTitle:  firstNonEmptyString(displayTitle, title),
		Overview:      extractHTML(javDBDescriptionPat, body),
		Year:          yearFromDate(extractText(javDBDatePattern, body)),
	}
}

func normalizeJavDBExternalID(externalID string) string {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return ""
	}
	if strings.HasPrefix(externalID, JavDBProvider+":") {
		return externalID
	}
	return JavDBProvider + ":" + normalizeJavDBPath(externalID)
}

func normalizeJavDBPath(path string) string {
	path = strings.TrimSpace(path)
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func extractText(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	if (len(match) == 2 || len(match) == 3) && (match[1] == "19" || match[1] == "20") {
		return match[0]
	}
	return strings.TrimSpace(match[len(match)-1])
}

func extractHTML(pattern *regexp.Regexp, value string) string {
	return stripTags(extractText(pattern, value))
}

func stripTags(value string) string {
	withoutTags := regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}

func absolutizeJavDBURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return defaultJavDBBaseURL + value
}
