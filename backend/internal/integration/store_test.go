package integration

import (
	"context"
	"testing"
)

func TestMemoryStoreCRUDAndRefreshPlan(t *testing.T) {
	store := NewMemoryStore()
	server, err := store.Create(context.Background(), ServerInput{
		Name:    "Emby",
		Type:    ServerEmby,
		BaseURL: "http://emby.local/",
		APIKey:  "secret",
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if server.BaseURL != "http://emby.local" {
		t.Fatalf("expected trimmed base url, got %q", server.BaseURL)
	}
	if !server.HasAPIKey {
		t.Fatal("expected api key presence")
	}
	key, ok, err := store.APIKey(context.Background(), server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || key != "secret" {
		t.Fatalf("expected stored key, got ok=%v key=%q", ok, key)
	}

	plan := BuildRefreshPlan(server, RefreshInput{LibraryID: "library-1"})
	if plan.Endpoint != "http://emby.local/Library/Refresh" {
		t.Fatalf("unexpected refresh endpoint %q", plan.Endpoint)
	}

	disabled := false
	updated, ok, err := store.Update(context.Background(), server.ID, ServerUpdate{Enabled: &disabled})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || updated.Enabled {
		t.Fatalf("expected disabled server, got %+v", updated)
	}
}
