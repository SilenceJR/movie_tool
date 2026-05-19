package scraper

import "testing"

func TestScoreCandidateInput(t *testing.T) {
	input := ScoreCandidate(ParsedMedia{
		Title: "Inception",
		Year:  2010,
	}, CandidateInput{
		Provider:   "tmdb",
		ExternalID: "27205",
		Title:      "Inception",
		Year:       2010,
	})

	if input.Score < 40 {
		t.Fatalf("expected useful score, got %d", input.Score)
	}
	if len(input.ScoreReasons) == 0 {
		t.Fatal("expected score reasons")
	}
}
