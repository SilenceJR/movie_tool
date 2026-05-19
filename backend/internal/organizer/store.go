package organizer

import (
	"context"
	"fmt"
	"sync"
)

type Store interface {
	SavePlan(context.Context, Plan) (Plan, error)
	GetPlan(context.Context, string) (Plan, bool, error)
	ListActions(context.Context, string) ([]Action, error)
}

type MemoryStore struct {
	mu      sync.Mutex
	plans   map[string]Plan
	actions map[string][]Action
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		plans:   make(map[string]Plan),
		actions: make(map[string][]Action),
	}
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
