package task

import "time"

type Type string

const (
	TypeLibraryScan       Type = "library_scan"
	TypeDownloadWatch     Type = "download_watch"
	TypeScrapeMedia       Type = "scrape_media"
	TypeDownloadImages    Type = "download_images"
	TypeTranslateMetadata Type = "translate_metadata"
	TypeOrganizeFiles     Type = "organize_files"
	TypeGenerateNFO       Type = "generate_nfo"
	TypeGenerateSTRM      Type = "generate_strm"
	TypeRefreshServer     Type = "refresh_server"
	TypeCleanupMissing    Type = "cleanup_missing"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type Task struct {
	ID        string    `json:"id"`
	Type      Type      `json:"type"`
	Status    Status    `json:"status"`
	Progress  int       `json:"progress"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogEntry struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
