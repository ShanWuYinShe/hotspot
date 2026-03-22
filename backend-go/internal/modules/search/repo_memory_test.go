package search

import (
	"context"
	"testing"
)

func TestInMemoryRepositorySuggestReturnsStubbedItems(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("unexpected result length: %#v", result)
	}
	if result.Items[0].Type != "tag" || result.Items[0].Payload["name"] != "hot" {
		t.Fatalf("unexpected first item: %#v", result.Items[0])
	}
}

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
