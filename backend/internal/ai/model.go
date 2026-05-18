package ai

type ProviderType string

const (
	ProviderOpenAICompatible ProviderType = "openai_compatible"
	ProviderOpenAI           ProviderType = "openai"
	ProviderGemini           ProviderType = "gemini"
	ProviderClaude           ProviderType = "claude"
	ProviderOllama           ProviderType = "ollama"
)

type Provider struct {
	ID           string
	Name         string
	Type         ProviderType
	BaseURL      string
	DefaultModel string
	Enabled      bool
}

type MatchSuggestion struct {
	CandidateID string   `json:"candidate_id"`
	Confidence  int      `json:"confidence"`
	Reason      string   `json:"reason"`
	Risks       []string `json:"risks"`
}
