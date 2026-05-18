package metadata

import "strings"

type MatchDecision string

const (
	DecisionMatched       MatchDecision = "matched"
	DecisionAmbiguous     MatchDecision = "ambiguous"
	DecisionLowConfidence MatchDecision = "low_confidence"
	DecisionUnmatched     MatchDecision = "unmatched"
)

func DecideMatch(score int, candidateCount int) MatchDecision {
	if candidateCount == 0 {
		return DecisionUnmatched
	}
	if score >= 90 && candidateCount == 1 {
		return DecisionMatched
	}
	if score >= 70 {
		return DecisionAmbiguous
	}
	return DecisionLowConfidence
}

type ScoreInput struct {
	ParsedTitle    string
	ParsedYear     int
	ParsedNumber   string
	CandidateTitle string
	CandidateYear  int
	CandidateID    string
}

type ScoreResult struct {
	Score   int
	Reasons []string
}

func ScoreCandidate(input ScoreInput) ScoreResult {
	score := 0
	var reasons []string

	if input.ParsedNumber != "" && strings.EqualFold(input.ParsedNumber, input.CandidateID) {
		score += 60
		reasons = append(reasons, "number_exact_match")
	}

	titleScore := titleSimilarity(input.ParsedTitle, input.CandidateTitle)
	if titleScore > 0 {
		score += titleScore
		reasons = append(reasons, "title_similarity")
	}

	if input.ParsedYear > 0 && input.CandidateYear > 0 {
		if input.ParsedYear == input.CandidateYear {
			score += 15
			reasons = append(reasons, "year_match")
		} else {
			score -= 20
			reasons = append(reasons, "year_conflict")
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return ScoreResult{Score: score, Reasons: reasons}
}

func titleSimilarity(left, right string) int {
	left = normalizeTitle(left)
	right = normalizeTitle(right)
	if left == "" || right == "" {
		return 0
	}
	if left == right {
		return 30
	}
	if strings.Contains(left, right) || strings.Contains(right, left) {
		return 20
	}
	leftParts := strings.Fields(left)
	rightSet := make(map[string]struct{})
	for _, part := range strings.Fields(right) {
		rightSet[part] = struct{}{}
	}
	matches := 0
	for _, part := range leftParts {
		if _, ok := rightSet[part]; ok {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return min(18, matches*6)
}

func normalizeTitle(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ", ":", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}
