package strm

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store interface {
	CreateRule(context.Context, RuleInput) (Rule, error)
	GetRule(context.Context, string) (Rule, bool, error)
	ListRules(context.Context) ([]Rule, error)
	UpdateRule(context.Context, string, RuleUpdate) (Rule, bool, error)
	DeleteRule(context.Context, string) (bool, error)
}

type MemoryStore struct {
	mu    sync.Mutex
	rules map[string]Rule
	now   func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rules: make(map[string]Rule), now: time.Now}
}

func (s *MemoryStore) CreateRule(_ context.Context, input RuleInput) (Rule, error) {
	if err := validateRule(input.Name, input.SourcePrefix, input.TargetPrefix, input.OutputPath); err != nil {
		return Rule{}, err
	}
	now := s.now().UTC()
	rule := Rule{
		ID:           fmt.Sprintf("strm_rule_%d", now.UnixNano()),
		Name:         input.Name,
		SourcePrefix: input.SourcePrefix,
		TargetPrefix: input.TargetPrefix,
		OutputPath:   input.OutputPath,
		Enabled:      input.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.ID] = rule
	return rule, nil
}

func (s *MemoryStore) GetRule(_ context.Context, id string) (Rule, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[id]
	return rule, ok, nil
}

func (s *MemoryStore) ListRules(context.Context) ([]Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules := make([]Rule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Name == rules[j].Name {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Name < rules[j].Name
	})
	return rules, nil
}

func (s *MemoryStore) UpdateRule(_ context.Context, id string, input RuleUpdate) (Rule, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[id]
	if !ok {
		return Rule{}, false, nil
	}
	applyRuleUpdate(&rule, input)
	if err := validateRule(rule.Name, rule.SourcePrefix, rule.TargetPrefix, rule.OutputPath); err != nil {
		return Rule{}, true, err
	}
	rule.UpdatedAt = s.now().UTC()
	s.rules[id] = rule
	return rule, true, nil
}

func (s *MemoryStore) DeleteRule(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return false, nil
	}
	delete(s.rules, id)
	return true, nil
}

func applyRuleUpdate(rule *Rule, input RuleUpdate) {
	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.SourcePrefix != nil {
		rule.SourcePrefix = *input.SourcePrefix
	}
	if input.TargetPrefix != nil {
		rule.TargetPrefix = *input.TargetPrefix
	}
	if input.OutputPath != nil {
		rule.OutputPath = *input.OutputPath
	}
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}
}
