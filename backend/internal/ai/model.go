package ai

import "time"

type ProviderType string

const (
	ProviderOpenAICompatible ProviderType = "openai_compatible"
	ProviderOpenAI           ProviderType = "openai"
	ProviderGemini           ProviderType = "gemini"
	ProviderClaude           ProviderType = "claude"
	ProviderOllama           ProviderType = "ollama"
)

type Provider struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         ProviderType `json:"provider_type"`
	BaseURL      string       `json:"base_url,omitempty"`
	DefaultModel string       `json:"default_model,omitempty"`
	Enabled      bool         `json:"enabled"`
	HasAPIKey    bool         `json:"has_api_key"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type MatchSuggestion struct {
	CandidateID string   `json:"candidate_id"`
	Confidence  int      `json:"confidence"`
	Reason      string   `json:"reason"`
	Risks       []string `json:"risks"`
}
