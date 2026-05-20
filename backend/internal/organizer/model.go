package organizer

import "time"

type ActionMode string

const (
	ActionMove     ActionMode = "move"
	ActionCopy     ActionMode = "copy"
	ActionHardlink ActionMode = "hardlink"
	ActionSymlink  ActionMode = "symlink"
)

type ConflictPolicy string

const (
	ConflictSkip                      ConflictPolicy = "skip"
	ConflictRename                    ConflictPolicy = "rename"
	ConflictOverwriteWithConfirmation ConflictPolicy = "overwrite_with_confirmation"
)

const (
	ConflictReasonDuplicateTargetPath = "duplicate target path in plan"
	ConflictReasonTargetPathExists    = "target path already exists"
	ConflictReasonOverwriteConfirmed  = "overwrite confirmed"
)

type PlanStatus string

const (
	PlanDraft     PlanStatus = "draft"
	PlanReady     PlanStatus = "ready"
	PlanRunning   PlanStatus = "running"
	PlanSucceeded PlanStatus = "succeeded"
	PlanFailed    PlanStatus = "failed"
	PlanCanceled  PlanStatus = "canceled"
)

type ActionStatus string

const (
	ActionPending    ActionStatus = "pending"
	ActionSkipped    ActionStatus = "skipped"
	ActionSucceeded  ActionStatus = "succeeded"
	ActionFailed     ActionStatus = "failed"
	ActionConflict   ActionStatus = "conflict"
	ActionRolledBack ActionStatus = "rolled_back"
)

type Rule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	LibraryID      string         `json:"library_id"`
	MediaType      string         `json:"media_type"`
	TargetRoot     string         `json:"target_root"`
	FolderTemplate string         `json:"folder_template"`
	FileTemplate   string         `json:"file_template"`
	SidecarPolicy  string         `json:"sidecar_policy"`
	ActionMode     ActionMode     `json:"action_mode"`
	ConflictPolicy ConflictPolicy `json:"conflict_policy"`
	Enabled        bool           `json:"enabled"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type RuleInput struct {
	Name           string         `json:"name"`
	LibraryID      string         `json:"library_id"`
	MediaType      string         `json:"media_type"`
	TargetRoot     string         `json:"target_root"`
	FolderTemplate string         `json:"folder_template"`
	FileTemplate   string         `json:"file_template"`
	SidecarPolicy  string         `json:"sidecar_policy"`
	ActionMode     ActionMode     `json:"action_mode"`
	ConflictPolicy ConflictPolicy `json:"conflict_policy"`
	Enabled        bool           `json:"enabled"`
}

type RuleUpdate struct {
	Name           *string         `json:"name"`
	LibraryID      *string         `json:"library_id"`
	MediaType      *string         `json:"media_type"`
	TargetRoot     *string         `json:"target_root"`
	FolderTemplate *string         `json:"folder_template"`
	FileTemplate   *string         `json:"file_template"`
	SidecarPolicy  *string         `json:"sidecar_policy"`
	ActionMode     *ActionMode     `json:"action_mode"`
	ConflictPolicy *ConflictPolicy `json:"conflict_policy"`
	Enabled        *bool           `json:"enabled"`
}

type Plan struct {
	ID        string     `json:"id"`
	LibraryID string     `json:"library_id"`
	Status    PlanStatus `json:"status"`
	DryRun    bool       `json:"dry_run"`
	Summary   Summary    `json:"summary"`
	Actions   []Action   `json:"actions"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Summary struct {
	TotalActions  int `json:"total_actions"`
	MoveCount     int `json:"move_count"`
	CopyCount     int `json:"copy_count"`
	LinkCount     int `json:"link_count"`
	ConflictCount int `json:"conflict_count"`
	SkipCount     int `json:"skip_count"`
}

type Action struct {
	ID             string       `json:"id"`
	PlanID         string       `json:"plan_id"`
	MediaID        string       `json:"media_id"`
	MediaFileID    string       `json:"media_file_id"`
	ActionType     ActionMode   `json:"action_type"`
	SourcePath     string       `json:"source_path"`
	TargetPath     string       `json:"target_path"`
	Status         ActionStatus `json:"status"`
	ConflictReason string       `json:"conflict_reason,omitempty"`
	Error          string       `json:"error,omitempty"`
	ExecutedAt     *time.Time   `json:"executed_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
}
