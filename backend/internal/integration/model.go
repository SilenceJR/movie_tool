package integration

import "time"

type ServerType string

const (
	ServerEmby     ServerType = "emby"
	ServerJellyfin ServerType = "jellyfin"
	ServerPlex     ServerType = "plex"
)

type Server struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      ServerType `json:"server_type"`
	BaseURL   string     `json:"base_url"`
	Enabled   bool       `json:"enabled"`
	HasAPIKey bool       `json:"has_api_key"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ServerInput struct {
	Name    string     `json:"name"`
	Type    ServerType `json:"server_type"`
	BaseURL string     `json:"base_url"`
	APIKey  string     `json:"api_key"`
	Enabled bool       `json:"enabled"`
}

type ServerUpdate struct {
	Name    *string     `json:"name"`
	Type    *ServerType `json:"server_type"`
	BaseURL *string     `json:"base_url"`
	APIKey  *string     `json:"api_key"`
	Enabled *bool       `json:"enabled"`
}

type RefreshInput struct {
	LibraryID string `json:"library_id"`
	Path      string `json:"path"`
}

type RefreshPlan struct {
	ServerID   string     `json:"server_id"`
	ServerType ServerType `json:"server_type"`
	BaseURL    string     `json:"base_url"`
	LibraryID  string     `json:"library_id,omitempty"`
	Path       string     `json:"path,omitempty"`
	Endpoint   string     `json:"endpoint"`
	Method     string     `json:"method"`
	Status     string     `json:"status"`
}
