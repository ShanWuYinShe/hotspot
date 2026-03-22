package feed

import (
	"context"
	"testing"
)

func TestInMemoryRepositoryListMomentsReturnsStubbedFeed(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListMoments(context.Background())
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0]["moment_id"] != "moment_feed_1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestInMemoryRepositoryListTrendingReturnsStubbedBundle(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListTrending(context.Background())
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(result.TrendingTags) != 1 || result.TrendingTags[0]["name"] != "hot" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.HotMoments) != 1 || result.HotMoments[0]["moment_id"] != "moment_hot_1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
