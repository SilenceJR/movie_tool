package localization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store interface {
	UpsertMetadata(context.Context, MetadataInput) (Metadata, error)
	ListMetadata(context.Context, MetadataQuery) ([]Metadata, error)
	Translate(context.Context, string, TranslateInput) (TranslationCache, *Metadata, error)
	ListTranslations(context.Context, string) ([]TranslationCache, error)
}

type MemoryStore struct {
	mu           sync.Mutex
	metadata     map[string]Metadata
	translations map[string]TranslationCache
	now          func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		metadata:     make(map[string]Metadata),
		translations: make(map[string]TranslationCache),
		now:          time.Now,
	}
}

func (s *MemoryStore) UpsertMetadata(_ context.Context, input MetadataInput) (Metadata, error) {
	if err := validateMetadata(input); err != nil {
		return Metadata{}, err
	}
	now := s.now().UTC()
	key := metadataKey(input.MediaID, input.Language, input.FieldName)
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.metadata[key]
	if item.ID == "" {
		item.ID = fmt.Sprintf("localized_%d", now.UnixNano())
		item.MediaID = input.MediaID
		item.Language = input.Language
		item.FieldName = input.FieldName
		item.CreatedAt = now
	}
	item.Value = input.Value
	item.Source = input.Source
	item.Provider = input.Provider
	item.Locked = input.Locked
	item.UpdatedAt = now
	s.metadata[key] = item
	return item, nil
}

func (s *MemoryStore) ListMetadata(_ context.Context, query MetadataQuery) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Metadata, 0, len(s.metadata))
	for _, item := range s.metadata {
		if query.MediaID != "" && item.MediaID != query.MediaID {
			continue
		}
		if query.Language != "" && item.Language != query.Language {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Language == items[j].Language {
			return items[i].FieldName < items[j].FieldName
		}
		return items[i].Language < items[j].Language
	})
	return items, nil
}

func (s *MemoryStore) Translate(ctx context.Context, mediaID string, input TranslateInput) (TranslationCache, *Metadata, error) {
	if err := validateTranslation(mediaID, input); err != nil {
		return TranslationCache{}, nil, err
	}
	now := s.now().UTC()
	hash := TextHash(input.SourceText)
	cacheKey := translationKey(input.SourceLanguage, input.TargetLanguage, hash)
	cache := TranslationCache{
		ID:             fmt.Sprintf("translation_%d", now.UnixNano()),
		SourceLanguage: input.SourceLanguage,
		TargetLanguage: input.TargetLanguage,
		SourceTextHash: hash,
		SourceText:     input.SourceText,
		TranslatedText: input.TranslatedText,
		Provider:       input.Provider,
		Model:          input.Model,
		Status:         statusOrDefault(input.Status),
		Confidence:     input.Confidence,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.mu.Lock()
	if existing := s.translations[cacheKey]; existing.ID != "" {
		cache.ID = existing.ID
		cache.CreatedAt = existing.CreatedAt
	}
	s.translations[cacheKey] = cache
	s.mu.Unlock()

	if !input.ApplyToMedia {
		return cache, nil, nil
	}
	metadata, err := s.UpsertMetadata(ctx, MetadataInput{
		MediaID:   mediaID,
		Language:  input.TargetLanguage,
		FieldName: input.FieldName,
		Value:     input.TranslatedText,
		Source:    "translation",
		Provider:  input.Provider,
	})
	if err != nil {
		return TranslationCache{}, nil, err
	}
	return cache, &metadata, nil
}

func (s *MemoryStore) ListTranslations(_ context.Context, _ string) ([]TranslationCache, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]TranslationCache, 0, len(s.translations))
	for _, item := range s.translations {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func TextHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validateMetadata(input MetadataInput) error {
	if input.MediaID == "" {
		return fmt.Errorf("media id is required")
	}
	if input.Language == "" {
		return fmt.Errorf("language is required")
	}
	if input.FieldName == "" {
		return fmt.Errorf("field name is required")
	}
	return nil
}

func validateTranslation(mediaID string, input TranslateInput) error {
	if mediaID == "" {
		return fmt.Errorf("media id is required")
	}
	if input.SourceLanguage == "" {
		return fmt.Errorf("source language is required")
	}
	if input.TargetLanguage == "" {
		return fmt.Errorf("target language is required")
	}
	if input.SourceText == "" {
		return fmt.Errorf("source text is required")
	}
	if input.ApplyToMedia && input.FieldName == "" {
		return fmt.Errorf("field name is required when applying translation")
	}
	return nil
}

func metadataKey(mediaID, language, fieldName string) string {
	return mediaID + "\x00" + language + "\x00" + fieldName
}

func translationKey(sourceLanguage, targetLanguage, sourceTextHash string) string {
	return sourceLanguage + "\x00" + targetLanguage + "\x00" + sourceTextHash
}

func statusOrDefault(status string) string {
	if status == "" {
		return "completed"
	}
	return status
}
