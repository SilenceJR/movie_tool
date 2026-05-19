package catalog

import (
	"context"
	"testing"
)

func TestMemoryStoreItemsAndVersions(t *testing.T) {
	store := NewMemoryStore()
	item, err := store.CreateItem(context.Background(), ItemInput{
		LibraryID:    "library-1",
		MediaType:    "movie",
		Title:        "Inception",
		DisplayTitle: "盗梦空间",
		Year:         2010,
		MatchStatus:  MatchStatusMatched,
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := store.ListItems(context.Background(), ItemQuery{Title: "盗梦", Year: 2010})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}

	first, err := store.CreateVersion(context.Background(), item.ID, VersionInput{
		Name:         "4K Remux",
		Resolution:   "2160p",
		QualityScore: 95,
		IsDefault:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateVersion(context.Background(), item.ID, VersionInput{
		Name:         "1080p Web",
		Resolution:   "1080p",
		QualityScore: 80,
	})
	if err != nil {
		t.Fatal(err)
	}

	selected, ok, err := store.SetDefaultVersion(context.Background(), item.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected default version update")
	}
	if selected.ID != second.ID || !selected.IsDefault {
		t.Fatalf("expected second version as default, got %+v", selected)
	}

	versions, err := store.ListVersions(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected two versions, got %d", len(versions))
	}
	if versions[0].ID != second.ID || !versions[0].IsDefault {
		t.Fatalf("expected second default version first, got %+v", versions[0])
	}
	if versions[1].ID != first.ID || versions[1].IsDefault {
		t.Fatalf("expected first version no longer default, got %+v", versions[1])
	}
}

func TestMemoryStorePeopleTagsCollections(t *testing.T) {
	store := NewMemoryStore()
	item, err := store.CreateItem(context.Background(), ItemInput{LibraryID: "library-1", MediaType: "av", Title: "Movie A"})
	if err != nil {
		t.Fatal(err)
	}
	person, err := store.CreatePerson(context.Background(), PersonInput{Name: "Actor A"})
	if err != nil {
		t.Fatal(err)
	}
	tag, err := store.CreateTag(context.Background(), TagInput{Name: "Series A", Category: "series"})
	if err != nil {
		t.Fatal(err)
	}
	collection, err := store.CreateCollection(context.Background(), CollectionInput{Name: "Collection A", Type: "series"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddMediaPerson(context.Background(), item.ID, MediaPersonInput{PersonID: person.ID, Role: "actor"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMediaTag(context.Background(), item.ID, MediaTagInput{TagID: tag.ID}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddCollectionItem(context.Background(), collection.ID, CollectionItemInput{MediaID: item.ID}); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListItems(context.Background(), ItemQuery{PersonID: person.ID, TagID: tag.ID, CollectionID: collection.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("expected linked item, got %+v", items)
	}
}

func TestMemoryStoreExternalIDsUpsertByProvider(t *testing.T) {
	store := NewMemoryStore()
	first, err := store.UpsertExternalID(context.Background(), ExternalIDInput{
		EntityType: "media",
		EntityID:   "media-1",
		Provider:   "tmdb",
		ExternalID: "27205",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UpsertExternalID(context.Background(), ExternalIDInput{
		EntityType: "media",
		EntityID:   "media-1",
		Provider:   "tmdb",
		ExternalID: "27206",
		URL:        "https://example.test/title/27206",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || !first.CreatedAt.Equal(second.CreatedAt) {
		t.Fatalf("expected provider upsert to keep identity, first=%+v second=%+v", first, second)
	}
	externalIDs, err := store.ListExternalIDs(context.Background(), "media", "media-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(externalIDs) != 1 || externalIDs[0].ExternalID != "27206" || externalIDs[0].URL == "" {
		t.Fatalf("expected updated external id, got %+v", externalIDs)
	}
}
