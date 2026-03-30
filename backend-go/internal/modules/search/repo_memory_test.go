package search

import (
	"context"
	"testing"
)

func TestInMemoryRepositorySuggestReturnsEmptyItemsForEmptyPrefix(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Suggest(context.Background(), SearchSuggestRequest{})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestInMemoryRepositorySuggestReturnsAllTypesByDefault(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("unexpected result length: %#v", result)
	}
	if result.Items[0].Type != "tag" || result.Items[1].Type != "user" || result.Items[2].Type != "group" || result.Items[3].Type != "moment" {
		t.Fatalf("unexpected item order: %#v", result.Items)
	}
}

func TestInMemoryRepositorySuggestFiltersTypes(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"group", "moment"}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("unexpected result length: %#v", result)
	}
	if result.Items[0].Type != "group" || result.Items[1].Type != "moment" {
		t.Fatalf("unexpected items: %#v", result.Items)
	}
}

func TestInMemoryRepositorySuggestFallsBackToAllTypesForUnknownTypes(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"weird"}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("unexpected result length: %#v", result)
	}
}
