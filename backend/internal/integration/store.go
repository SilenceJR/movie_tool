package integration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Create(context.Context, ServerInput) (Server, error)
	Get(context.Context, string) (Server, bool, error)
	List(context.Context) ([]Server, error)
	Update(context.Context, string, ServerUpdate) (Server, bool, error)
	Delete(context.Context, string) (bool, error)
	APIKey(context.Context, string) (string, bool, error)
}

type MemoryStore struct {
	mu      sync.Mutex
	servers map[string]Server
	apiKeys map[string]string
	now     func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		servers: make(map[string]Server),
		apiKeys: make(map[string]string),
		now:     time.Now,
	}
}

func (s *MemoryStore) Create(_ context.Context, input ServerInput) (Server, error) {
	server, err := serverFromInput(input, s.now().UTC())
	if err != nil {
		return Server{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	server.ID = fmt.Sprintf("integration_%d", server.CreatedAt.UnixNano())
	if input.APIKey != "" {
		server.HasAPIKey = true
		s.apiKeys[server.ID] = input.APIKey
	}
	s.servers[server.ID] = server
	return server, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Server, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	server, ok := s.servers[id]
	return server, ok, nil
}

func (s *MemoryStore) List(context.Context) ([]Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	servers := make([]Server, 0, len(s.servers))
	for _, server := range s.servers {
		servers = append(servers, server)
	}
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Name == servers[j].Name {
			return servers[i].ID < servers[j].ID
		}
		return servers[i].Name < servers[j].Name
	})
	return servers, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, input ServerUpdate) (Server, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	server, ok := s.servers[id]
	if !ok {
		return Server{}, false, nil
	}
	applyUpdate(&server, input)
	if err := validateServer(server.Name, server.Type, server.BaseURL); err != nil {
		return Server{}, true, err
	}
	if input.APIKey != nil {
		if *input.APIKey == "" {
			delete(s.apiKeys, id)
			server.HasAPIKey = false
		} else {
			s.apiKeys[id] = *input.APIKey
			server.HasAPIKey = true
		}
	}
	server.UpdatedAt = s.now().UTC()
	s.servers[id] = server
	return server, true, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.servers[id]; !ok {
		return false, nil
	}
	delete(s.servers, id)
	delete(s.apiKeys, id)
	return true, nil
}

func (s *MemoryStore) APIKey(_ context.Context, id string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.apiKeys[id]
	return key, ok, nil
}

func BuildRefreshPlan(server Server, input RefreshInput) RefreshPlan {
	endpoint := "/Library/Refresh"
	if server.Type == ServerPlex {
		endpoint = "/library/sections/all/refresh"
	}
	return RefreshPlan{
		ServerID:   server.ID,
		ServerType: server.Type,
		BaseURL:    strings.TrimRight(server.BaseURL, "/"),
		LibraryID:  input.LibraryID,
		Path:       input.Path,
		Endpoint:   strings.TrimRight(server.BaseURL, "/") + endpoint,
		Method:     "POST",
		Status:     "planned",
	}
}

func serverFromInput(input ServerInput, now time.Time) (Server, error) {
	if err := validateServer(input.Name, input.Type, input.BaseURL); err != nil {
		return Server{}, err
	}
	return Server{
		Name:      input.Name,
		Type:      input.Type,
		BaseURL:   strings.TrimRight(input.BaseURL, "/"),
		Enabled:   input.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func validateServer(name string, serverType ServerType, baseURL string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("integration name is required")
	}
	if serverType == "" {
		return fmt.Errorf("server type is required")
	}
	switch serverType {
	case ServerEmby, ServerJellyfin, ServerPlex:
	default:
		return fmt.Errorf("unsupported server type %q", serverType)
	}
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("base url is required")
	}
	return nil
}

func applyUpdate(server *Server, input ServerUpdate) {
	if input.Name != nil {
		server.Name = *input.Name
	}
	if input.Type != nil {
		server.Type = *input.Type
	}
	if input.BaseURL != nil {
		server.BaseURL = strings.TrimRight(*input.BaseURL, "/")
	}
	if input.Enabled != nil {
		server.Enabled = *input.Enabled
	}
}
