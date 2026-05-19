package scraper

import "movie-tool/backend/internal/metadata"

type ParsedMedia struct {
	Title  string
	Year   int
	Number string
}

func ScoreCandidate(parsed ParsedMedia, candidate CandidateInput) CandidateInput {
	result := metadata.ScoreCandidate(metadata.ScoreInput{
		ParsedTitle:    parsed.Title,
		ParsedYear:     parsed.Year,
		ParsedNumber:   parsed.Number,
		CandidateTitle: candidate.Title,
		CandidateYear:  candidate.Year,
		CandidateID:    candidate.ExternalID,
	})
	candidate.Score = result.Score
	candidate.ScoreReasons = result.Reasons
	return candidate
}
