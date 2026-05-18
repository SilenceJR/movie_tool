package library

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreCreateListUpdateDelete(t *testing.T) {
	store := NewMemoryStore()
	store.now = func() time.Time {
		return time.Unix(1, 0)
	}

	created, err := store.Create(context.Background(), CreateInput{
		Name:      "Movies",
		MediaType: MediaTypeMovie,
		Path:      "/media/movies",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Language != "zh-CN" {
		t.Fatalf("expected default zh-CN language, got %q", created.Language)
	}

	libraries, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(libraries) != 1 {
		t.Fatalf("expected 1 library, got %d", len(libraries))
	}

	name := "Films"
	updated, ok, err := store.Update(context.Background(), created.ID, UpdateInput{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || updated.Name != "Films" {
		t.Fatalf("expected updated library, got ok=%v name=%q", ok, updated.Name)
	}

	deleted, err := store.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected delete to succeed")
	}
}
