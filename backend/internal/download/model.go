package download

import "time"

type Directory struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	LibraryID       string    `json:"library_id"`
	MediaType       string    `json:"media_type,omitempty"`
	ActionMode      string    `json:"action_mode"`
	OrganizerRuleID string    `json:"organizer_rule_id,omitempty"`
	Enabled         bool      `json:"enabled"`
	WatchEnabled    bool      `json:"watch_enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DirectoryInput struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	LibraryID       string `json:"library_id"`
	MediaType       string `json:"media_type"`
	ActionMode      string `json:"action_mode"`
	OrganizerRuleID string `json:"organizer_rule_id"`
	Enabled         bool   `json:"enabled"`
	WatchEnabled    bool   `json:"watch_enabled"`
}

type DirectoryUpdate struct {
	Name            *string `json:"name"`
	Path            *string `json:"path"`
	LibraryID       *string `json:"library_id"`
	MediaType       *string `json:"media_type"`
	ActionMode      *string `json:"action_mode"`
	OrganizerRuleID *string `json:"organizer_rule_id"`
	Enabled         *bool   `json:"enabled"`
	WatchEnabled    *bool   `json:"watch_enabled"`
}
