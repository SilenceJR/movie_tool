package ai

import (
	"context"
	"testing"
)

func TestMemoryStoreProviderCRUD(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.CreateProvider(context.Background(), ProviderInput{
		Name:         "OpenAI",
		Type:         ProviderOpenAI,
		APIKey:       "secret",
		DefaultModel: "gpt-4.1-mini",
		Enabled:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.HasAPIKey {
		t.Fatal("expected provider to report api key presence")
	}

	key, ok, err := store.APIKey(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || key != "secret" {
		t.Fatalf("expected stored api key, got ok=%v key=%q", ok, key)
	}

	model := "gpt-4.1"
	emptyKey := ""
	updated, ok, err := store.UpdateProvider(context.Background(), created.ID, ProviderUpdate{
		DefaultModel: &model,
		APIKey:       &emptyKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected provider update")
	}
	if updated.DefaultModel != model {
		t.Fatalf("expected model update, got %q", updated.DefaultModel)
	}
	if updated.HasAPIKey {
		t.Fatal("expected api key to be cleared")
	}
}
