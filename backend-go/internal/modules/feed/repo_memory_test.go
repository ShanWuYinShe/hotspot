package feed

import "testing"

func TestInMemoryRepositoryListMomentsReturnsTypedStub(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListMoments(nil, momentsQuery{})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].MomentID != "moment_feed_1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestInMemoryRepositoryListTrendingReturnsTypedBundle(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListTrending(nil, "")
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(result.TrendingTags) != 1 || result.TrendingTags[0].Name != "hot" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.HotMoments) != 1 || result.HotMoments[0].MomentID != "moment_hot_1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
