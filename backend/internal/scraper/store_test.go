package scraper

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreCandidates(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	low, err := store.CreateCandidate(context.Background(), CandidateInput{
		MediaFileID:  "file-1",
		Provider:     "tmdb",
		ExternalID:   "1",
		Title:        "Wrong",
		Score:        20,
		ScoreReasons: []string{"title_similarity"},
	})
	if err != nil {
		t.Fatal(err)
	}
	high, err := store.CreateCandidate(context.Background(), CandidateInput{
		MediaFileID:  "file-1",
		Provider:     "tmdb",
		ExternalID:   "2",
		Title:        "Inception",
		Score:        95,
		ScoreReasons: []string{"title_similarity", "year_match"},
	})
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := store.ListCandidates(context.Background(), CandidateQuery{MediaFileID: "file-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].ID != high.ID || candidates[1].ID != low.ID {
		t.Fatalf("expected candidates sorted by score desc, got %+v", candidates)
	}
}

func TestMemoryStoreDecisions(t *testing.T) {
	store := NewMemoryStore()

	decision, err := store.CreateDecision(context.Background(), DecisionInput{
		MediaID:        "media-1",
		CandidateID:    "candidate-1",
		DecisionSource: DecisionSourceUser,
		Decision:       DecisionSelect,
		Confidence:     95,
		Reason:         "用户确认",
		Locked:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Locked {
		t.Fatal("expected locked decision")
	}

	decisions, err := store.ListDecisions(context.Background(), DecisionQuery{MediaID: "media-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0].CandidateID != "candidate-1" {
		t.Fatalf("expected candidate-1, got %q", decisions[0].CandidateID)
	}
}
