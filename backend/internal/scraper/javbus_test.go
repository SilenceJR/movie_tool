package scraper

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestJavBusSearchParsesCandidates(t *testing.T) {
	client := JavBusClient{
		BaseURL: "https://javbus.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/search/SSNI-00123" {
				t.Fatalf("unexpected search request %s", r.URL.String())
			}
			return htmlResponse(`
				<a class="movie-box" href="/SSNI-00123">
					<img src="/pics/ssni.jpg">
					<div class="photo-info">SSNI-00123 Example Title</div>
					<div>2020-02-03</div>
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
	if candidate.Provider != "javbus" || candidate.ExternalID != "javbus:/SSNI-00123" || candidate.Number != "SSNI-00123" || candidate.Year != 2020 || candidate.Score != 90 {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
	if candidate.PosterURL != "https://www.javbus.com/pics/ssni.jpg" {
		t.Fatalf("unexpected poster url %q", candidate.PosterURL)
	}
}

func TestJavBusFetchParsesMetadata(t *testing.T) {
	client := JavBusClient{
		BaseURL: "https://javbus.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/SSNI-00123" {
				t.Fatalf("unexpected fetch request %s", r.URL.String())
			}
			return htmlResponse(`
				<h3>SSNI-00123 Example Title</h3>
				<img src="/pics/ssni-detail.jpg">
				<p><span>發行日期:</span> 2020-02-03</p>
				<p><span>長度:</span> 120分鐘</p>
				<p><span>製作商:</span><a>Example Studio</a></p>
				<p><span>系列:</span><a>Example Series</a></p>
				<p><span>演員:</span><a>Alice</a><a>Bob</a></p>
				<a class="star-name">Alice</a>
				<span class="genre"><a>Drama</a></span>
				<span class="genre"><a>高清</a></span>
			`), nil
		})},
	}

	metadata, err := client.FetchByID(context.Background(), "javbus:/SSNI-00123")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Provider != "javbus" || metadata.ExternalID != "javbus:/SSNI-00123" || metadata.Number != "SSNI-00123" || metadata.Title != "SSNI-00123" || metadata.Year != 2020 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if metadata.DisplayTitle != "Example Title" || metadata.PosterURL != "https://www.javbus.com/pics/ssni-detail.jpg" || metadata.RuntimeMinutes != 120 || metadata.Studio != "Example Studio" || metadata.Series != "Example Series" {
		t.Fatalf("unexpected metadata structured fields: %#v", metadata)
	}
	if strings.Join(metadata.Actors, ",") != "Alice,Bob" {
		t.Fatalf("unexpected metadata actors: %#v", metadata)
	}
	if strings.Join(metadata.Tags, ",") != "Drama,高清" {
		t.Fatalf("unexpected metadata tags: %#v", metadata)
	}
}
