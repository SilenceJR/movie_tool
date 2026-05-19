package organizer

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
	SavePlan(context.Context, Plan) (Plan, error)
	UpdatePlan(context.Context, Plan) (Plan, error)
	GetPlan(context.Context, string) (Plan, bool, error)
	ListActions(context.Context, string) ([]Action, error)
}

type MemoryStore struct {
	mu      sync.Mutex
	rules   map[string]Rule
	plans   map[string]Plan
	actions map[string][]Action
	now     func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rules:   make(map[string]Rule),
		plans:   make(map[string]Plan),
		actions: make(map[string][]Action),
		now:     time.Now,
	}
}

func (s *MemoryStore) CreateRule(_ context.Context, input RuleInput) (Rule, error) {
	rule, err := ruleFromInput(input, s.now().UTC())
	if err != nil {
		return Rule{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rule.ID = fmt.Sprintf("organizer_rule_%d", rule.CreatedAt.UnixNano())
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
	if err := validateRule(rule); err != nil {
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

func (s *MemoryStore) SavePlan(_ context.Context, plan Plan) (Plan, error) {
	if plan.ID == "" {
		return Plan{}, fmt.Errorf("plan id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	copied := clonePlan(plan)
	s.plans[copied.ID] = copied
	s.actions[copied.ID] = append([]Action(nil), copied.Actions...)
	return clonePlan(copied), nil
}

func (s *MemoryStore) UpdatePlan(_ context.Context, plan Plan) (Plan, error) {
	if plan.ID == "" {
		return Plan{}, fmt.Errorf("plan id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.plans[plan.ID]; !ok {
		return Plan{}, fmt.Errorf("organizer plan not found")
	}
	copied := clonePlan(plan)
	s.plans[copied.ID] = copied
	s.actions[copied.ID] = append([]Action(nil), copied.Actions...)
	return clonePlan(copied), nil
}

func (s *MemoryStore) GetPlan(_ context.Context, id string) (Plan, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, ok := s.plans[id]
	if !ok {
		return Plan{}, false, nil
	}
	plan.Actions = append([]Action(nil), s.actions[id]...)
	return clonePlan(plan), true, nil
}

func (s *MemoryStore) ListActions(_ context.Context, planID string) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Action(nil), s.actions[planID]...), nil
}

func clonePlan(plan Plan) Plan {
	plan.Actions = append([]Action(nil), plan.Actions...)
	return plan
}

func ruleFromInput(input RuleInput, now time.Time) (Rule, error) {
	rule := Rule{
		Name:           input.Name,
		LibraryID:      input.LibraryID,
		MediaType:      input.MediaType,
		TargetRoot:     input.TargetRoot,
		FolderTemplate: input.FolderTemplate,
		FileTemplate:   input.FileTemplate,
		SidecarPolicy:  defaultString(input.SidecarPolicy, "include"),
		ActionMode:     defaultActionMode(input.ActionMode),
		ConflictPolicy: defaultConflictPolicy(input.ConflictPolicy),
		Enabled:        input.Enabled,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateRule(rule); err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func validateRule(rule Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("organizer rule name is required")
	}
	if rule.TargetRoot == "" {
		return fmt.Errorf("target root is required")
	}
	if rule.SidecarPolicy == "" {
		return fmt.Errorf("sidecar policy is required")
	}
	if rule.ActionMode == "" {
		return fmt.Errorf("action mode is required")
	}
	if rule.ConflictPolicy == "" {
		return fmt.Errorf("conflict policy is required")
	}
	return nil
}

func applyRuleUpdate(rule *Rule, input RuleUpdate) {
	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.LibraryID != nil {
		rule.LibraryID = *input.LibraryID
	}
	if input.MediaType != nil {
		rule.MediaType = *input.MediaType
	}
	if input.TargetRoot != nil {
		rule.TargetRoot = *input.TargetRoot
	}
	if input.FolderTemplate != nil {
		rule.FolderTemplate = *input.FolderTemplate
	}
	if input.FileTemplate != nil {
		rule.FileTemplate = *input.FileTemplate
	}
	if input.SidecarPolicy != nil {
		rule.SidecarPolicy = *input.SidecarPolicy
	}
	if input.ActionMode != nil {
		rule.ActionMode = *input.ActionMode
	}
	if input.ConflictPolicy != nil {
		rule.ConflictPolicy = *input.ConflictPolicy
	}
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}
}

func defaultConflictPolicy(policy ConflictPolicy) ConflictPolicy {
	if policy == "" {
		return ConflictSkip
	}
	return policy
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
