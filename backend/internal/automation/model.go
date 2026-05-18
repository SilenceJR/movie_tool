package automation

import "time"

type Type string

const (
	TypeScanLibrary      Type = "scan_library"
	TypeScrapePending    Type = "scrape_pending"
	TypeTranslateMissing Type = "translate_missing"
	TypeOrganizeFiles    Type = "organize_files"
	TypeGenerateNFO      Type = "generate_nfo"
	TypeGenerateSTRM     Type = "generate_strm"
	TypeRefreshServer    Type = "refresh_server"
	TypeCleanupMissing   Type = "cleanup_missing"
)

type ScheduleType string

const (
	ScheduleInterval ScheduleType = "interval"
	ScheduleCron     ScheduleType = "cron"
)

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunCanceled  RunStatus = "canceled"
)

type Automation struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         Type         `json:"automation_type"`
	ScheduleType ScheduleType `json:"schedule_type"`
	Schedule     string       `json:"schedule"`
	Scope        Scope        `json:"scope"`
	Options      Options      `json:"options"`
	Enabled      bool         `json:"enabled"`
	LastRunAt    *time.Time   `json:"last_run_at,omitempty"`
	NextRunAt    *time.Time   `json:"next_run_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type Scope struct {
	LibraryID string   `json:"library_id,omitempty"`
	MediaType string   `json:"media_type,omitempty"`
	MediaIDs  []string `json:"media_ids,omitempty"`
	Query     string   `json:"query,omitempty"`
}

type Options struct {
	DryRun      bool   `json:"dry_run,omitempty"`
	RetryLimit  int    `json:"retry_limit,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
	QuietWindow string `json:"quiet_window,omitempty"`
}

type Run struct {
	ID           string     `json:"id"`
	AutomationID string     `json:"automation_id"`
	TaskID       string     `json:"task_id"`
	Status       RunStatus  `json:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
