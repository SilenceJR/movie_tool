package localization

import "time"

type Metadata struct {
	ID        string    `json:"id"`
	MediaID   string    `json:"media_id"`
	Language  string    `json:"language"`
	FieldName string    `json:"field_name"`
	Value     string    `json:"value"`
	Source    string    `json:"source,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Locked    bool      `json:"locked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TranslationCache struct {
	ID             string    `json:"id"`
	SourceLanguage string    `json:"source_language"`
	TargetLanguage string    `json:"target_language"`
	SourceTextHash string    `json:"source_text_hash"`
	SourceText     string    `json:"source_text"`
	TranslatedText string    `json:"translated_text,omitempty"`
	Provider       string    `json:"provider,omitempty"`
	Model          string    `json:"model,omitempty"`
	Status         string    `json:"status"`
	Confidence     int       `json:"confidence,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MetadataInput struct {
	MediaID   string `json:"media_id"`
	Language  string `json:"language"`
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
	Source    string `json:"source"`
	Provider  string `json:"provider"`
	Locked    bool   `json:"locked"`
}

type MetadataQuery struct {
	MediaID  string
	Language string
}

type TranslateInput struct {
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
	FieldName      string `json:"field_name"`
	SourceText     string `json:"source_text"`
	TranslatedText string `json:"translated_text"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Status         string `json:"status"`
	Confidence     int    `json:"confidence"`
	ApplyToMedia   bool   `json:"apply_to_media"`
}
