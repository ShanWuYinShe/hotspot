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

func TestInMemoryRepositoryDismissIsIdempotent(t *testing.T) {
	repo := NewInMemoryRepository()
	cmd := dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "moment_feed_1"}

	if err := repo.Dismiss(nil, cmd); err != nil {
		t.Fatalf("first dismiss returned error: %v", err)
	}
	if err := repo.Dismiss(nil, cmd); err != nil {
		t.Fatalf("second dismiss returned error: %v", err)
	}
}

func TestInMemoryRepositoryDismissReturnsNotFoundForMissingTarget(t *testing.T) {
	repo := NewInMemoryRepository()

	if err := repo.Dismiss(nil, dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "missing"}); err != ErrDismissTargetNotFound {
		t.Fatalf("expected ErrDismissTargetNotFound, got %v", err)
	}
}

func TestInMemoryRepositoryDismissRejectsInvalidCommand(t *testing.T) {
	repo := NewInMemoryRepository()

	if err := repo.Dismiss(nil, dismissCommand{}); err == nil {
		t.Fatal("expected error")
	}
}
