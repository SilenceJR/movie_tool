package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	yearPattern      = regexp.MustCompile(`(?i)(?:^|[ ._\-\(\[])(19\d{2}|20\d{2})(?:$|[ ._\-\)\]])`)
	episodePattern   = regexp.MustCompile(`(?i)s(\d{1,2})e(\d{1,3})`)
	resolutionTokens = []string{"4320p", "2160p", "1080p", "720p", "480p"}
	sourceTokens     = []string{"remux", "bluray", "blu-ray", "web-dl", "webdl", "webrip", "hdtv", "dvdrip"}
	videoTokens      = []string{"h265", "h.265", "hevc", "x265", "h264", "h.264", "avc", "x264", "av1"}
	audioTokens      = []string{"truehd", "dts-hd", "dts", "atmos", "aac", "flac", "ac3"}
	hdrTokens        = []string{"dolby vision", "dovi", "dv", "hdr10+", "hdr10", "hdr", "sdr"}
	subtitleTokens   = []string{"chs", "cht", "chinese", "subtitle", "subbed", "中字", "字幕", "简中", "繁中"}
	avNumberPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bFC2(?:-PPV)?[-_ ]?(\d{5,})\b`),
		regexp.MustCompile(`(?i)\b([A-Z]{2,8})[-_ ]?(\d{2,6})\b`),
		regexp.MustCompile(`(?i)\b(HEYZO)[-_ ]?(\d{3,6})\b`),
		regexp.MustCompile(`(?i)\b(CARIB)[-_ ]?(\d{6}[-_ ]?\d{3})\b`),
	}
)

type ParsedFile struct {
	LibraryID    string    `json:"library_id"`
	LibraryName  string    `json:"library_name"`
	MediaType    string    `json:"media_type"`
	Path         string    `json:"path"`
	FileName     string    `json:"file_name"`
	Extension    string    `json:"extension"`
	BaseName     string    `json:"base_name"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	Title        string    `json:"title"`
	Year         int       `json:"year"`
	Season       int       `json:"season"`
	Episode      int       `json:"episode"`
	Number       string    `json:"number"`
	Resolution   string    `json:"resolution"`
	Source       string    `json:"source"`
	VideoCodec   string    `json:"video_codec"`
	AudioCodec   string    `json:"audio_codec"`
	HDRFormat    string    `json:"hdr_format"`
	Subtitles    []string  `json:"subtitles"`
	ReleaseGroup string    `json:"release_group"`
}

func ParseFile(path string) ParsedFile {
	fileName := filepath.Base(path)
	extension := strings.ToLower(filepath.Ext(fileName))
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	normalized := normalizeSeparators(baseName)
	lower := strings.ToLower(normalized)

	parsed := ParsedFile{
		Path:      path,
		FileName:  fileName,
		Extension: extension,
		BaseName:  baseName,
	}

	parsed.Year = parseYear(normalized)
	parsed.Season, parsed.Episode = parseEpisode(normalized)
	parsed.Number = parseAVNumber(normalized)
	parsed.Resolution = firstToken(lower, resolutionTokens)
	parsed.Source = canonicalSource(firstToken(lower, sourceTokens))
	parsed.VideoCodec = canonicalVideo(firstToken(lower, videoTokens))
	parsed.AudioCodec = strings.ToUpper(firstToken(lower, audioTokens))
	parsed.HDRFormat = canonicalHDR(firstToken(lower, hdrTokens))
	parsed.Subtitles = parseSubtitles(lower)
	parsed.ReleaseGroup = parseReleaseGroup(baseName)
	parsed.Title = parseTitle(normalized, parsed)

	return parsed
}

func parseYear(value string) int {
	matches := yearPattern.FindStringSubmatch(value)
	if len(matches) < 2 {
		return 0
	}
	year, _ := strconv.Atoi(matches[1])
	return year
}

func parseEpisode(value string) (int, int) {
	matches := episodePattern.FindStringSubmatch(value)
	if len(matches) < 3 {
		return 0, 0
	}
	season, _ := strconv.Atoi(matches[1])
	episode, _ := strconv.Atoi(matches[2])
	return season, episode
}

func parseAVNumber(value string) string {
	for _, pattern := range avNumberPatterns {
		matches := pattern.FindStringSubmatch(value)
		if len(matches) == 2 {
			return "FC2-PPV-" + matches[1]
		}
		if len(matches) >= 3 {
			return strings.ToUpper(matches[1]) + "-" + strings.ReplaceAll(matches[2], "_", "-")
		}
	}
	return ""
}

func parseTitle(value string, parsed ParsedFile) string {
	if parsed.Number != "" && strings.HasPrefix(strings.ToLower(value), strings.ToLower(parsed.Number)) {
		return parsed.Number
	}

	title := value
	if parsed.Season > 0 && parsed.Episode > 0 {
		episodeToken := "s" + leftPad(parsed.Season) + "e" + leftPad(parsed.Episode)
		idx := strings.Index(strings.ToLower(title), episodeToken)
		if idx > 0 {
			title = title[:idx]
		}
	}
	cutTokens := []string{parsed.Number, parsed.Resolution, parsed.Source, parsed.VideoCodec, parsed.AudioCodec, parsed.HDRFormat}
	for _, token := range cutTokens {
		if token == "" {
			continue
		}
		idx := strings.Index(strings.ToLower(title), strings.ToLower(token))
		if idx > 0 {
			title = title[:idx]
		}
	}
	if parsed.Year > 0 {
		idx := strings.Index(title, strconv.Itoa(parsed.Year))
		if idx > 0 {
			title = title[:idx]
		}
	}
	title = strings.Trim(title, " ._-[]()")
	if title == "" && parsed.Number != "" {
		return parsed.Number
	}
	return title
}

func leftPad(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func parseSubtitles(lower string) []string {
	var result []string
	for _, token := range subtitleTokens {
		if strings.Contains(lower, token) {
			result = append(result, token)
		}
	}
	return result
}

func parseReleaseGroup(baseName string) string {
	if idx := strings.LastIndex(baseName, "-"); idx > 0 && idx < len(baseName)-1 {
		group := strings.TrimSpace(baseName[idx+1:])
		if group != "" && !strings.Contains(group, " ") {
			return group
		}
	}
	return ""
}

func normalizeSeparators(value string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func firstToken(lower string, tokens []string) string {
	for _, token := range tokens {
		if strings.Contains(lower, token) {
			return token
		}
	}
	return ""
}

func canonicalSource(value string) string {
	switch value {
	case "blu-ray":
		return "bluray"
	case "webdl":
		return "web-dl"
	default:
		return value
	}
}

func canonicalVideo(value string) string {
	switch value {
	case "h.265", "h265", "x265":
		return "hevc"
	case "h.264", "h264", "x264":
		return "avc"
	default:
		return value
	}
}

func canonicalHDR(value string) string {
	switch value {
	case "dolby vision", "dovi", "dv":
		return "dolby_vision"
	case "hdr10+":
		return "hdr10_plus"
	default:
		return value
	}
}
