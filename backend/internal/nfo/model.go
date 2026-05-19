package nfo

type MediaInfo struct {
	ID            string   `json:"id,omitempty"`
	MediaType     string   `json:"media_type"`
	Title         string   `json:"title,omitempty"`
	OriginalTitle string   `json:"original_title,omitempty"`
	DisplayTitle  string   `json:"display_title,omitempty"`
	Year          int      `json:"year,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	Runtime       int      `json:"runtime,omitempty"`
	ReleaseDate   string   `json:"release_date,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Genres        []string `json:"genres,omitempty"`
}

type MetadataField struct {
	Language  string `json:"language"`
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
}

type GenerateRequest struct {
	Media     MediaInfo       `json:"media"`
	Metadata  []MetadataField `json:"metadata"`
	Language  string          `json:"language"`
	FileName  string          `json:"file_name"`
	OutputDir string          `json:"output_dir"`
	DryRun    bool            `json:"dry_run"`
}

type Entry struct {
	MediaID    string `json:"media_id,omitempty"`
	FileName   string `json:"file_name"`
	OutputPath string `json:"output_path,omitempty"`
	Content    string `json:"content"`
	Status     string `json:"status"`
}

type Plan struct {
	DryRun  bool    `json:"dry_run"`
	Entries []Entry `json:"entries"`
	Count   int     `json:"count"`
}
