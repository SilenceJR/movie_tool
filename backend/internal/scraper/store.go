package scraper

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type StoredCandidate struct {
	ID            string    `json:"id"`
	MediaFileID   string    `json:"media_file_id,omitempty"`
	MediaID       string    `json:"media_id,omitempty"`
	Provider      string    `json:"provider"`
	ExternalID    string    `json:"external_id"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"original_title"`
	Year          int       `json:"year"`
	PosterURL     string    `json:"poster_url"`
	Overview      string    `json:"overview"`
	Score         int       `json:"score"`
	ScoreReasons  []string  `json:"score_reasons"`
	RawPayload    string    `json:"raw_payload,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CandidateInput struct {
	MediaFileID   string   `json:"media_file_id"`
	MediaID       string   `json:"media_id"`
	Provider      string   `json:"provider"`
	ExternalID    string   `json:"external_id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Year          int      `json:"year"`
	PosterURL     string   `json:"poster_url"`
	Overview      string   `json:"overview"`
	Score         int      `json:"score"`
	ScoreReasons  []string `json:"score_reasons"`
	RawPayload    string   `json:"raw_payload"`
}

type Store interface {
	CreateCandidate(context.Context, CandidateInput) (StoredCandidate, error)
	ListCandidates(context.Context, CandidateQuery) ([]StoredCandidate, error)
	CreateDecision(context.Context, DecisionInput) (Decision, error)
	ListDecisions(context.Context, DecisionQuery) ([]Decision, error)
}

type CandidateQuery struct {
	MediaFileID string
	MediaID     string
}

type MemoryStore struct {
	mu         sync.Mutex
	candidates map[string]StoredCandidate
	decisions  map[string]Decision
	now        func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		candidates: make(map[string]StoredCandidate),
		decisions:  make(map[string]Decision),
		now:        time.Now,
	}
}

func (s *MemoryStore) CreateCandidate(_ context.Context, input CandidateInput) (StoredCandidate, error) {
	if input.Provider == "" {
		return StoredCandidate{}, fmt.Errorf("candidate provider is required")
	}
	if input.ExternalID == "" {
		return StoredCandidate{}, fmt.Errorf("candidate external id is required")
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate := StoredCandidate{
		ID:            fmt.Sprintf("candidate_%d_%d", now.UnixNano(), len(s.candidates)+1),
		MediaFileID:   input.MediaFileID,
		MediaID:       input.MediaID,
		Provider:      input.Provider,
		ExternalID:    input.ExternalID,
		Title:         input.Title,
		OriginalTitle: input.OriginalTitle,
		Year:          input.Year,
		PosterURL:     input.PosterURL,
		Overview:      input.Overview,
		Score:         input.Score,
		ScoreReasons:  append([]string(nil), input.ScoreReasons...),
		RawPayload:    input.RawPayload,
		CreatedAt:     now,
	}
	s.candidates[candidate.ID] = candidate
	return cloneCandidate(candidate), nil
}

func (s *MemoryStore) ListCandidates(_ context.Context, query CandidateQuery) ([]StoredCandidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidates := make([]StoredCandidate, 0)
	for _, candidate := range s.candidates {
		if query.MediaFileID != "" && candidate.MediaFileID != query.MediaFileID {
			continue
		}
		if query.MediaID != "" && candidate.MediaID != query.MediaID {
			continue
		}
		candidates = append(candidates, cloneCandidate(candidate))
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates, nil
}

func cloneCandidate(candidate StoredCandidate) StoredCandidate {
	candidate.ScoreReasons = append([]string(nil), candidate.ScoreReasons...)
	return candidate
}

type DecisionSource string

const (
	DecisionSourceUser   DecisionSource = "user"
	DecisionSourceSystem DecisionSource = "system"
	DecisionSourceAI     DecisionSource = "ai"
)

type DecisionValue string

const (
	DecisionSelect DecisionValue = "select"
	DecisionIgnore DecisionValue = "ignore"
	DecisionManual DecisionValue = "manual"
)

type Decision struct {
	ID             string         `json:"id"`
	MediaID        string         `json:"media_id"`
	CandidateID    string         `json:"candidate_id,omitempty"`
	DecisionSource DecisionSource `json:"decision_source"`
	Decision       DecisionValue  `json:"decision"`
	Confidence     int            `json:"confidence"`
	Reason         string         `json:"reason,omitempty"`
	Locked         bool           `json:"locked"`
	CreatedAt      time.Time      `json:"created_at"`
}

type DecisionInput struct {
	MediaID        string         `json:"media_id"`
	CandidateID    string         `json:"candidate_id"`
	DecisionSource DecisionSource `json:"decision_source"`
	Decision       DecisionValue  `json:"decision"`
	Confidence     int            `json:"confidence"`
	Reason         string         `json:"reason"`
	Locked         bool           `json:"locked"`
}

type DecisionQuery struct {
	MediaID string
}

func (s *MemoryStore) CreateDecision(_ context.Context, input DecisionInput) (Decision, error) {
	if input.MediaID == "" {
		return Decision{}, fmt.Errorf("media id is required")
	}
	if input.DecisionSource == "" {
		input.DecisionSource = DecisionSourceUser
	}
	if input.Decision == "" {
		input.Decision = DecisionSelect
	}
	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	decision := Decision{
		ID:             fmt.Sprintf("decision_%d_%d", now.UnixNano(), len(s.decisions)+1),
		MediaID:        input.MediaID,
		CandidateID:    input.CandidateID,
		DecisionSource: input.DecisionSource,
		Decision:       input.Decision,
		Confidence:     input.Confidence,
		Reason:         input.Reason,
		Locked:         input.Locked,
		CreatedAt:      now,
	}
	s.decisions[decision.ID] = decision
	return decision, nil
}

func (s *MemoryStore) ListDecisions(_ context.Context, query DecisionQuery) ([]Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	decisions := make([]Decision, 0)
	for _, decision := range s.decisions {
		if query.MediaID != "" && decision.MediaID != query.MediaID {
			continue
		}
		decisions = append(decisions, decision)
	}
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].CreatedAt.Before(decisions[j].CreatedAt)
	})
	return decisions, nil
}
