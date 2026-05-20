package scraper

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestJavDBSearchParsesCandidates(t *testing.T) {
	client := JavDBClient{
		BaseURL: "https://javdb.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/search" || r.URL.Query().Get("q") != "SSNI-00123" || r.URL.Query().Get("f") != "all" {
				t.Fatalf("unexpected search request %s", r.URL.String())
			}
			return htmlResponse(`
				<a class="box" href="/v/javdb-id">
					<img src="/covers/ssni.jpg">
					<div class="video-title">SSNI-00123 Example Title</div>
					<div class="meta">2020-02-03</div>
				</a>
			`), nil
		})},
	}

	candidates, err := client.Search(context.Background(), SearchQuery{MediaType: "av", Number: "ssni00123"})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", candidates)
	}
	candidate := candidates[0]
	if candidate.ExternalID != "javdb:/v/javdb-id" || candidate.Number != "SSNI-00123" || candidate.Year != 2020 || candidate.Score != 90 {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
	if candidate.PosterURL != "https://javdb.com/covers/ssni.jpg" {
		t.Fatalf("unexpected poster url %q", candidate.PosterURL)
	}
}

func TestJavDBFetchParsesMetadata(t *testing.T) {
	client := JavDBClient{
		BaseURL: "https://javdb.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v/javdb-id" {
				t.Fatalf("unexpected fetch request %s", r.URL.String())
			}
			return htmlResponse(`
				<h2>SSNI-00123 Example Title</h2>
				<img src="/covers/ssni-detail.jpg">
				<div class="release-date">2020-02-03</div>
				<div class="description">Example overview</div>
				<div class="field"><strong>日期:</strong><span>2020-02-03</span></div>
				<div class="field"><strong>時長:</strong><span>120 分鐘</span></div>
				<div class="field"><strong>片商:</strong><a>Example Studio</a></div>
				<div class="field"><strong>系列:</strong><a>Example Series</a></div>
				<div class="field"><strong>演員:</strong><a>Alice</a><a>Bob</a></div>
				<div class="field"><strong>類別:</strong><a>Drama</a><a>高清</a></div>
			`), nil
		})},
	}

	metadata, err := client.FetchByID(context.Background(), "javdb:/v/javdb-id")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Provider != "javdb" || metadata.ExternalID != "javdb:/v/javdb-id" || metadata.Number != "SSNI-00123" || metadata.Title != "SSNI-00123" || metadata.Year != 2020 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if metadata.DisplayTitle != "Example Title" || metadata.Overview != "Example overview" || metadata.PosterURL != "https://javdb.com/covers/ssni-detail.jpg" {
		t.Fatalf("unexpected metadata text fields: %#v", metadata)
	}
	if metadata.ReleaseDate != "2020-02-03" || metadata.RuntimeMinutes != 120 || metadata.Studio != "Example Studio" || metadata.Series != "Example Series" {
		t.Fatalf("unexpected metadata structured fields: %#v", metadata)
	}
	if strings.Join(metadata.Actors, ",") != "Alice,Bob" || strings.Join(metadata.Tags, ",") != "Drama,高清" {
		t.Fatalf("unexpected metadata people/tags: %#v", metadata)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func htmlResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
