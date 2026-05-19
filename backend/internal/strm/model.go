package strm

import "time"

type Rule struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SourcePrefix string    `json:"source_prefix"`
	TargetPrefix string    `json:"target_prefix"`
	OutputPath   string    `json:"output_path"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RuleInput struct {
	Name         string `json:"name"`
	SourcePrefix string `json:"source_prefix"`
	TargetPrefix string `json:"target_prefix"`
	OutputPath   string `json:"output_path"`
	Enabled      bool   `json:"enabled"`
}

type RuleUpdate struct {
	Name         *string `json:"name"`
	SourcePrefix *string `json:"source_prefix"`
	TargetPrefix *string `json:"target_prefix"`
	OutputPath   *string `json:"output_path"`
	Enabled      *bool   `json:"enabled"`
}

type FileInfo struct {
	ID        string `json:"id,omitempty"`
	LibraryID string `json:"library_id,omitempty"`
	Path      string `json:"path"`
	FileName  string `json:"file_name,omitempty"`
	Status    string `json:"file_status,omitempty"`
}

type GenerateRequest struct {
	RuleID    string     `json:"rule_id"`
	LibraryID string     `json:"library_id"`
	Files     []FileInfo `json:"files"`
	DryRun    bool       `json:"dry_run"`
}

type Entry struct {
	MediaFileID string `json:"media_file_id,omitempty"`
	SourcePath  string `json:"source_path"`
	TargetURL   string `json:"target_url"`
	OutputPath  string `json:"output_path"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type Plan struct {
	RuleID    string  `json:"rule_id"`
	LibraryID string  `json:"library_id,omitempty"`
	DryRun    bool    `json:"dry_run"`
	Entries   []Entry `json:"entries"`
	Count     int     `json:"count"`
}

type ValidateRequest struct {
	Rule RuleInput `json:"rule"`
	Path string    `json:"path"`
}

type ValidationResult struct {
	Valid      bool   `json:"valid"`
	TargetURL  string `json:"target_url,omitempty"`
	OutputPath string `json:"output_path,omitempty"`
	Error      string `json:"error,omitempty"`
}
