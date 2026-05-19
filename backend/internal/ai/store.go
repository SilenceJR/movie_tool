package ai

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store interface {
	CreateProvider(context.Context, ProviderInput) (Provider, error)
	GetProvider(context.Context, string) (Provider, bool, error)
	ListProviders(context.Context) ([]Provider, error)
	UpdateProvider(context.Context, string, ProviderUpdate) (Provider, bool, error)
	DeleteProvider(context.Context, string) (bool, error)
	APIKey(context.Context, string) (string, bool, error)
}

type ProviderInput struct {
	Name         string       `json:"name"`
	Type         ProviderType `json:"provider_type"`
	BaseURL      string       `json:"base_url"`
	APIKey       string       `json:"api_key"`
	DefaultModel string       `json:"default_model"`
	Enabled      bool         `json:"enabled"`
}

type ProviderUpdate struct {
	Name         *string       `json:"name"`
	Type         *ProviderType `json:"provider_type"`
	BaseURL      *string       `json:"base_url"`
	APIKey       *string       `json:"api_key"`
	DefaultModel *string       `json:"default_model"`
	Enabled      *bool         `json:"enabled"`
}

type MemoryStore struct {
	mu      sync.Mutex
	items   map[string]Provider
	apiKeys map[string]string
	now     func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items:   make(map[string]Provider),
		apiKeys: make(map[string]string),
		now:     time.Now,
	}
}

func (s *MemoryStore) CreateProvider(_ context.Context, input ProviderInput) (Provider, error) {
	provider, err := providerFromInput(input, s.now().UTC())
	if err != nil {
		return Provider{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	provider.ID = fmt.Sprintf("ai_provider_%d", provider.CreatedAt.UnixNano())
	if input.APIKey != "" {
		provider.HasAPIKey = true
		s.apiKeys[provider.ID] = input.APIKey
	}
	s.items[provider.ID] = provider
	return provider, nil
}

func (s *MemoryStore) GetProvider(_ context.Context, id string) (Provider, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.items[id]
	return provider, ok, nil
}

func (s *MemoryStore) ListProviders(context.Context) ([]Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Provider, 0, len(s.items))
	for _, provider := range s.items {
		items = append(items, provider)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].ID < items[j].ID
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (s *MemoryStore) UpdateProvider(_ context.Context, id string, input ProviderUpdate) (Provider, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.items[id]
	if !ok {
		return Provider{}, false, nil
	}
	applyProviderUpdate(&provider, input)
	if input.APIKey != nil {
		if *input.APIKey == "" {
			delete(s.apiKeys, id)
			provider.HasAPIKey = false
		} else {
			s.apiKeys[id] = *input.APIKey
			provider.HasAPIKey = true
		}
	}
	provider.UpdatedAt = s.now().UTC()
	s.items[id] = provider
	return provider, true, nil
}

func (s *MemoryStore) DeleteProvider(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false, nil
	}
	delete(s.items, id)
	delete(s.apiKeys, id)
	return true, nil
}

func (s *MemoryStore) APIKey(_ context.Context, id string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.apiKeys[id]
	return key, ok, nil
}

func providerFromInput(input ProviderInput, now time.Time) (Provider, error) {
	if input.Name == "" {
		return Provider{}, fmt.Errorf("provider name is required")
	}
	if input.Type == "" {
		return Provider{}, fmt.Errorf("provider type is required")
	}
	return Provider{
		Name:         input.Name,
		Type:         input.Type,
		BaseURL:      input.BaseURL,
		DefaultModel: input.DefaultModel,
		Enabled:      input.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func applyProviderUpdate(provider *Provider, input ProviderUpdate) {
	if input.Name != nil {
		provider.Name = *input.Name
	}
	if input.Type != nil {
		provider.Type = *input.Type
	}
	if input.BaseURL != nil {
		provider.BaseURL = *input.BaseURL
	}
	if input.DefaultModel != nil {
		provider.DefaultModel = *input.DefaultModel
	}
	if input.Enabled != nil {
		provider.Enabled = *input.Enabled
	}
}
