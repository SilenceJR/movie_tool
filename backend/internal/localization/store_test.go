package localization

import (
	"context"
	"testing"
)

func TestMemoryStoreTranslateAndApplyMetadata(t *testing.T) {
	store := NewMemoryStore()
	cache, metadata, err := store.Translate(context.Background(), "media-1", TranslateInput{
		SourceLanguage: "ja-JP",
		TargetLanguage: "zh-CN",
		FieldName:      "overview",
		SourceText:     "Japanese overview",
		TranslatedText: "中文简介",
		Provider:       "manual",
		Status:         "completed",
		Confidence:     90,
		ApplyToMedia:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cache.SourceTextHash != TextHash("Japanese overview") {
		t.Fatalf("unexpected hash %q", cache.SourceTextHash)
	}
	if metadata == nil {
		t.Fatal("expected applied metadata")
	}
	if metadata.Language != "zh-CN" || metadata.Value != "中文简介" {
		t.Fatalf("unexpected metadata %+v", metadata)
	}

	items, err := store.ListMetadata(context.Background(), MetadataQuery{MediaID: "media-1", Language: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one metadata item, got %d", len(items))
	}
}

func TestMemoryStoreUpsertMetadata(t *testing.T) {
	store := NewMemoryStore()
	first, err := store.UpsertMetadata(context.Background(), MetadataInput{
		MediaID:   "media-1",
		Language:  "zh-CN",
		FieldName: "title",
		Value:     "旧标题",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UpsertMetadata(context.Background(), MetadataInput{
		MediaID:   "media-1",
		Language:  "zh-CN",
		FieldName: "title",
		Value:     "新标题",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same metadata id after upsert, got %q and %q", first.ID, second.ID)
	}
	if second.Value != "新标题" {
		t.Fatalf("expected updated value, got %q", second.Value)
	}
}
