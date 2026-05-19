package scraper

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type SQLStore struct {
	db  SQLDB
	now func() time.Time
}

func NewSQLStore(db SQLDB) *SQLStore {
	return &SQLStore{db: db, now: time.Now}
}

func (s *SQLStore) CreateCandidate(ctx context.Context, input CandidateInput) (StoredCandidate, error) {
	if input.Provider == "" {
		return StoredCandidate{}, fmt.Errorf("candidate provider is required")
	}
	if input.ExternalID == "" {
		return StoredCandidate{}, fmt.Errorf("candidate external id is required")
	}
	now := s.now().UTC()
	candidate := StoredCandidate{
		ID:            newID("candidate"),
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

	reasonsJSON, err := json.Marshal(candidate.ScoreReasons)
	if err != nil {
		return StoredCandidate{}, err
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO scrape_candidates (
  id, media_file_id, media_id, provider, external_id, title, original_title, year,
  poster_url, overview, score, score_reasons, raw_payload, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		candidate.ID,
		nullableString(candidate.MediaFileID),
		nullableString(candidate.MediaID),
		candidate.Provider,
		candidate.ExternalID,
		candidate.Title,
		candidate.OriginalTitle,
		candidate.Year,
		candidate.PosterURL,
		candidate.Overview,
		candidate.Score,
		string(reasonsJSON),
		candidate.RawPayload,
		formatTime(candidate.CreatedAt),
	)
	if err != nil {
		return StoredCandidate{}, err
	}
	return candidate, nil
}

func (s *SQLStore) ListCandidates(ctx context.Context, query CandidateQuery) ([]StoredCandidate, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 2)
	if query.MediaFileID != "" {
		where += " AND media_file_id = ?"
		args = append(args, query.MediaFileID)
	}
	if query.MediaID != "" {
		where += " AND media_id = ?"
		args = append(args, query.MediaID)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, media_file_id, media_id, provider, external_id, title, original_title, year,
       poster_url, overview, score, score_reasons, raw_payload, created_at
FROM scrape_candidates
`+where+`
ORDER BY score DESC, created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []StoredCandidate
	for rows.Next() {
		candidate, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (s *SQLStore) CreateDecision(ctx context.Context, input DecisionInput) (Decision, error) {
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
	decision := Decision{
		ID:             newID("decision"),
		MediaID:        input.MediaID,
		CandidateID:    input.CandidateID,
		DecisionSource: input.DecisionSource,
		Decision:       input.Decision,
		Confidence:     input.Confidence,
		Reason:         input.Reason,
		Locked:         input.Locked,
		CreatedAt:      now,
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO scrape_decisions (
  id, media_id, candidate_id, decision_source, decision, confidence, reason, locked, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		decision.ID,
		decision.MediaID,
		nullableString(decision.CandidateID),
		string(decision.DecisionSource),
		string(decision.Decision),
		decision.Confidence,
		decision.Reason,
		boolInt(decision.Locked),
		formatTime(decision.CreatedAt),
	)
	if err != nil {
		return Decision{}, err
	}
	return decision, nil
}

func (s *SQLStore) ListDecisions(ctx context.Context, query DecisionQuery) ([]Decision, error) {
	where := "WHERE 1 = 1"
	args := make([]any, 0, 1)
	if query.MediaID != "" {
		where += " AND media_id = ?"
		args = append(args, query.MediaID)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, media_id, candidate_id, decision_source, decision, confidence, reason, locked, created_at
FROM scrape_decisions
`+where+`
ORDER BY created_at ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return decisions, nil
}

type candidateScanner interface {
	Scan(dest ...any) error
}

func scanCandidate(scanner candidateScanner) (StoredCandidate, error) {
	var candidate StoredCandidate
	var mediaFileID sql.NullString
	var mediaID sql.NullString
	var reasonsJSON sql.NullString
	var rawPayload sql.NullString
	var createdAt string
	err := scanner.Scan(
		&candidate.ID,
		&mediaFileID,
		&mediaID,
		&candidate.Provider,
		&candidate.ExternalID,
		&candidate.Title,
		&candidate.OriginalTitle,
		&candidate.Year,
		&candidate.PosterURL,
		&candidate.Overview,
		&candidate.Score,
		&reasonsJSON,
		&rawPayload,
		&createdAt,
	)
	if err != nil {
		return StoredCandidate{}, err
	}
	candidate.MediaFileID = mediaFileID.String
	candidate.MediaID = mediaID.String
	candidate.RawPayload = rawPayload.String
	if reasonsJSON.Valid && reasonsJSON.String != "" {
		if err := json.Unmarshal([]byte(reasonsJSON.String), &candidate.ScoreReasons); err != nil {
			return StoredCandidate{}, err
		}
	}
	candidate.CreatedAt = parseTime(createdAt)
	return candidate, nil
}

func scanDecision(scanner candidateScanner) (Decision, error) {
	var decision Decision
	var candidateID sql.NullString
	var decisionSource string
	var decisionValue string
	var reason sql.NullString
	var locked int
	var createdAt string
	err := scanner.Scan(
		&decision.ID,
		&decision.MediaID,
		&candidateID,
		&decisionSource,
		&decisionValue,
		&decision.Confidence,
		&reason,
		&locked,
		&createdAt,
	)
	if err != nil {
		return Decision{}, err
	}
	decision.CandidateID = candidateID.String
	decision.DecisionSource = DecisionSource(decisionSource)
	decision.Decision = DecisionValue(decisionValue)
	decision.Reason = reason.String
	decision.Locked = locked == 1
	decision.CreatedAt = parseTime(createdAt)
	return decision, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func newID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
