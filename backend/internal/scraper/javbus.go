package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	JavBusProvider       = "javbus"
	defaultJavBusBaseURL = "https://www.javbus.com"
)

type JavBusClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c JavBusClient) Name() string {
	return JavBusProvider
}

func (c JavBusClient) Supports(mediaType string) bool {
	return strings.EqualFold(strings.TrimSpace(mediaType), "av")
}

func (c JavBusClient) Search(ctx context.Context, query SearchQuery) ([]Candidate, error) {
	number, ok := ParseAVNumber(firstNonEmptyString(query.Number, query.Title))
	if !ok {
		return nil, fmt.Errorf("av number could not be parsed")
	}
	body, err := c.get(ctx, "/search/"+url.PathEscape(number.Normalized), nil)
	if err != nil {
		return nil, err
	}
	return parseJavBusSearch(body, number), nil
}

func (c JavBusClient) Fetch(ctx context.Context, candidate Candidate) (*Metadata, error) {
	return c.FetchByID(ctx, candidate.ExternalID)
}

func (c JavBusClient) FetchByID(ctx context.Context, externalID string) (*Metadata, error) {
	path := strings.TrimSpace(strings.TrimPrefix(externalID, JavBusProvider+":"))
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
	metadata := parseJavBusDetail(body, externalID)
	if metadata.ExternalID == "" {
		metadata.ExternalID = JavBusProvider + ":" + path
	}
	return metadata, nil
}

func (c JavBusClient) get(ctx context.Context, path string, values url.Values) (string, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultJavBusBaseURL
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
		return "", fmt.Errorf("javbus request failed: status=%d", response.StatusCode)
	}
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

var (
	javBusCardPattern    = regexp.MustCompile(`(?is)<a[^>]+class=["'][^"']*movie-box[^"']*["'][^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	javBusTitlePattern   = regexp.MustCompile(`(?is)<[^>]+class=["'][^"']*(?:photo-info|title)[^"']*["'][^>]*>(.*?)</[^>]+>`)
	javBusImagePattern   = regexp.MustCompile(`(?is)<img[^>]+(?:src|data-src)=["']([^"']+)["']`)
	javBusDatePattern    = regexp.MustCompile(`\b(19|20)\d{2}[-/]\d{1,2}[-/]\d{1,2}\b`)
	javBusH3Pattern      = regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	javBusNumberPattern  = regexp.MustCompile(`(?i)\b([A-Z]{2,8}-\d{2,6}|FC2-PPV-\d{5,}|HEYZO-\d{3,}|(?:CARIB|1PONDO|10MUSUME)-\d{6}-\d{2,3})\b`)
	javBusGenrePat       = regexp.MustCompile(`(?is)<span[^>]+class=["'][^"']*genre[^"']*["'][^>]*>(.*?)</span>`)
	javBusAnchorPattern  = regexp.MustCompile(`(?is)<a[^>]*>(.*?)</a>`)
	javBusStudioPattern  = regexp.MustCompile(`(?is)(?:製作商|制作商|片商|Studio)[：:\s]*</span>\s*<a[^>]*>(.*?)</a>`)
	javBusSeriesPattern  = regexp.MustCompile(`(?is)(?:系列|Series)[：:\s]*</span>\s*<a[^>]*>(.*?)</a>`)
	javBusRuntimePattern = regexp.MustCompile(`(?is)(?:長度|长度|Runtime)[：:\s]*</span>\s*(\d{1,4})`)
	javBusActorBlockPat  = regexp.MustCompile(`(?is)(?:演員|演员|Actress|Actor|Cast)[：:\s]*</span>(.*?)</p>`)
	javBusStarLinkPat    = regexp.MustCompile(`(?is)<a[^>]+class=["'][^"']*(?:star-name|avatar-box|star-box)[^"']*["'][^>]*>(.*?)</a>`)
)

func parseJavBusSearch(body string, number AVNumber) []Candidate {
	matches := javBusCardPattern.FindAllStringSubmatch(body, -1)
	candidates := make([]Candidate, 0, len(matches))
	for _, match := range matches {
		path := normalizeJavBusPath(match[1])
		card := match[2]
		title := firstNonEmptyString(extractHTML(javBusTitlePattern, card), stripTags(card))
		if title == "" {
			continue
		}
		candidate := Candidate{
			Provider:      JavBusProvider,
			ExternalID:    JavBusProvider + ":" + path,
			Number:        number.Normalized,
			Title:         title,
			OriginalTitle: title,
			Year:          yearFromDate(extractText(javBusDatePattern, card)),
			PosterURL:     absolutizeJavBusURL(extractText(javBusImagePattern, card)),
		}
		if strings.Contains(strings.ToUpper(title), number.Normalized) {
			candidate.Score = 90
			candidate.ScoreReasons = []string{"番号精确匹配"}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func parseJavBusDetail(body string, externalID string) *Metadata {
	title := extractHTML(javBusH3Pattern, body)
	number := strings.ToUpper(extractText(javBusNumberPattern, body))
	releaseDate := extractText(javBusDatePattern, body)
	displayTitle := strings.TrimSpace(strings.TrimPrefix(title, number))
	tags := extractJavBusAnchors(javBusGenrePat.FindAllString(body, -1))
	return &Metadata{
		Provider:       JavBusProvider,
		ExternalID:     normalizeJavBusExternalID(externalID),
		Number:         number,
		Title:          firstNonEmptyString(number, title),
		OriginalTitle:  title,
		DisplayTitle:   firstNonEmptyString(displayTitle, title),
		Year:           yearFromDate(releaseDate),
		PosterURL:      absolutizeJavBusURL(extractText(javBusImagePattern, body)),
		ReleaseDate:    releaseDate,
		RuntimeMinutes: parseMinutes(extractText(javBusRuntimePattern, body) + "分"),
		Studio:         extractHTML(javBusStudioPattern, body),
		Series:         extractHTML(javBusSeriesPattern, body),
		Actors:         extractJavBusActors(body),
		Tags:           tags,
	}
}

func normalizeJavBusExternalID(externalID string) string {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return ""
	}
	if strings.HasPrefix(externalID, JavBusProvider+":") {
		return externalID
	}
	return JavBusProvider + ":" + normalizeJavBusPath(externalID)
}

func normalizeJavBusPath(path string) string {
	path = strings.TrimSpace(path)
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func extractJavBusAnchors(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		matches := javBusAnchorPattern.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			text := stripTags(match[1])
			if text != "" {
				result = append(result, text)
			}
		}
	}
	return uniqueStrings(result)
}

func extractJavBusActors(body string) []string {
	values := extractJavBusAnchors(javBusActorBlockPat.FindAllString(body, -1))
	for _, match := range javBusStarLinkPat.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		text := stripTags(match[1])
		if text != "" {
			values = append(values, text)
		}
	}
	return uniqueStrings(values)
}

func absolutizeJavBusURL(value string) string {
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
	return defaultJavBusBaseURL + value
}
