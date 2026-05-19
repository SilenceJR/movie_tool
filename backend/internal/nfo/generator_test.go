package nfo

import (
	"strings"
	"testing"
)

func TestGeneratorUsesLocalizedMetadata(t *testing.T) {
	plan, err := NewGenerator().Generate(GenerateRequest{
		Media: MediaInfo{
			ID:            "media-1",
			MediaType:     "movie",
			Title:         "Inception",
			OriginalTitle: "Inception",
			Year:          2010,
			Overview:      "English plot",
		},
		Metadata: []MetadataField{
			{Language: "zh-CN", FieldName: "title", Value: "盗梦空间"},
			{Language: "zh-CN", FieldName: "overview", Value: "中文简介"},
		},
		Language:  "zh-CN",
		OutputDir: "/library/Inception",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Count != 1 {
		t.Fatalf("expected one entry, got %d", plan.Count)
	}
	entry := plan.Entries[0]
	if entry.OutputPath != "/library/Inception/movie.nfo" {
		t.Fatalf("unexpected output path %q", entry.OutputPath)
	}
	if !strings.Contains(entry.Content, "<title>盗梦空间</title>") {
		t.Fatalf("expected localized title in nfo:\n%s", entry.Content)
	}
	if !strings.Contains(entry.Content, "<plot>中文简介</plot>") {
		t.Fatalf("expected localized plot in nfo:\n%s", entry.Content)
	}
	if !strings.Contains(entry.Content, "<year>2010</year>") {
		t.Fatalf("expected year in nfo:\n%s", entry.Content)
	}
}

func TestGeneratorTVShowRoot(t *testing.T) {
	plan, err := NewGenerator().Generate(GenerateRequest{
		Media: MediaInfo{MediaType: "tv", Title: "Show"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].FileName != "tvshow.nfo" {
		t.Fatalf("expected tvshow.nfo, got %q", plan.Entries[0].FileName)
	}
	if !strings.Contains(plan.Entries[0].Content, "<tvshow>") {
		t.Fatalf("expected tvshow root:\n%s", plan.Entries[0].Content)
	}
}
