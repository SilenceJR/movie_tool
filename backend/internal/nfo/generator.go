package nfo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

type Generator struct{}

func NewGenerator() Generator {
	return Generator{}
}

func (Generator) Generate(request GenerateRequest) (Plan, error) {
	if request.Media.MediaType == "" {
		return Plan{}, fmt.Errorf("media type is required")
	}
	if request.Language == "" {
		request.Language = "zh-CN"
	}
	if request.FileName == "" {
		request.FileName = defaultFileName(request.Media.MediaType)
	}
	content, err := buildXML(request)
	if err != nil {
		return Plan{}, err
	}
	outputPath := request.FileName
	if request.OutputDir != "" {
		outputPath = filepath.Join(request.OutputDir, request.FileName)
	}
	entry := Entry{
		MediaID:    request.Media.ID,
		FileName:   request.FileName,
		OutputPath: outputPath,
		Content:    content,
		Status:     "planned",
	}
	return Plan{DryRun: true, Entries: []Entry{entry}, Count: 1}, nil
}

type nfoDocument struct {
	XMLName       xml.Name
	Title         string   `xml:"title,omitempty"`
	OriginalTitle string   `xml:"originaltitle,omitempty"`
	SortTitle     string   `xml:"sorttitle,omitempty"`
	Year          int      `xml:"year,omitempty"`
	Plot          string   `xml:"plot,omitempty"`
	Runtime       int      `xml:"runtime,omitempty"`
	Premiered     string   `xml:"premiered,omitempty"`
	Genre         []string `xml:"genre,omitempty"`
	Tag           []string `xml:"tag,omitempty"`
}

func buildXML(request GenerateRequest) (string, error) {
	media := applyLocalizedMetadata(request.Media, request.Metadata, request.Language)
	document := nfoDocument{
		XMLName:       xml.Name{Local: rootName(media.MediaType)},
		Title:         firstNonEmpty(media.DisplayTitle, media.Title, media.OriginalTitle),
		OriginalTitle: media.OriginalTitle,
		SortTitle:     firstNonEmpty(media.Title, media.DisplayTitle, media.OriginalTitle),
		Year:          media.Year,
		Plot:          media.Overview,
		Runtime:       media.Runtime,
		Premiered:     media.ReleaseDate,
		Genre:         media.Genres,
		Tag:           media.Tags,
	}
	var buffer bytes.Buffer
	buffer.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buffer)
	encoder.Indent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return "", err
	}
	buffer.WriteByte('\n')
	return buffer.String(), nil
}

func applyLocalizedMetadata(media MediaInfo, metadata []MetadataField, language string) MediaInfo {
	for _, field := range metadata {
		if field.Language != language || field.Value == "" {
			continue
		}
		switch field.FieldName {
		case "title", "display_title":
			media.DisplayTitle = field.Value
		case "overview", "plot":
			media.Overview = field.Value
		case "original_title":
			media.OriginalTitle = field.Value
		}
	}
	return media
}

func defaultFileName(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "tv", "series", "anime":
		return "tvshow.nfo"
	default:
		return "movie.nfo"
	}
}

func rootName(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "tv", "series", "anime":
		return "tvshow"
	case "episode":
		return "episodedetails"
	default:
		return "movie"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
