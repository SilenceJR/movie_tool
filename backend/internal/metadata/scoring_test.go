package metadata

import "testing"

func TestScoreCandidateNumberExact(t *testing.T) {
	result := ScoreCandidate(ScoreInput{
		ParsedTitle:    "ABC-123",
		ParsedNumber:   "ABC-123",
		CandidateTitle: "ABC-123",
		CandidateID:    "ABC-123",
	})

	if result.Score < 90 {
		t.Fatalf("expected high score, got %d", result.Score)
	}
}

func TestScoreCandidateYearConflict(t *testing.T) {
	result := ScoreCandidate(ScoreInput{
		ParsedTitle:    "Inception",
		ParsedYear:     2010,
		CandidateTitle: "Inception",
		CandidateYear:  2012,
	})

	if result.Score >= 30 {
		t.Fatalf("expected year conflict to reduce score, got %d", result.Score)
	}
}

func TestDecideMatchAmbiguous(t *testing.T) {
	decision := DecideMatch(80, 2)
	if decision != DecisionAmbiguous {
		t.Fatalf("expected ambiguous, got %s", decision)
	}
}
