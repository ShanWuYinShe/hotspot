package feed

import "testing"

func TestInMemoryRepositoryListMomentsReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListMoments(nil, momentsQuery{})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(result.Items))
	}
}

func TestInMemoryRepositoryListTrendingReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListTrending(nil, "")
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(result.TrendingTags) != 0 {
		t.Fatalf("expected empty trending tags, got %d", len(result.TrendingTags))
	}
	if len(result.HotMoments) != 0 {
		t.Fatalf("expected empty hot moments, got %d", len(result.HotMoments))
	}
}

func TestInMemoryRepositoryDismissIsIdempotent(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.RegisterTestTarget("moment", "test_moment")
	cmd := dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "test_moment"}

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

func TestInMemoryRepositoryListGroupsReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListGroups(nil, groupsQuery{})
	if err != nil {
		t.Fatalf("ListGroups returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty groups, got %d", len(result.Items))
	}
}

func TestInMemoryRepositoryListUsersReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListUsers(nil, usersQuery{})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty users, got %d", len(result.Items))
	}
}

func TestInMemoryRepositoryDismissWithVariousTargetTypes(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.RegisterTestTarget("user", "u1")
	repo.RegisterTestTarget("group", "g1")

	if err := repo.Dismiss(nil, dismissCommand{ViewerUserID: "viewer", TargetType: "user", TargetID: "u1"}); err != nil {
		t.Fatalf("dismiss user failed: %v", err)
	}
	if err := repo.Dismiss(nil, dismissCommand{ViewerUserID: "viewer", TargetType: "group", TargetID: "g1"}); err != nil {
		t.Fatalf("dismiss group failed: %v", err)
	}
	if err := repo.Dismiss(nil, dismissCommand{ViewerUserID: "viewer", TargetType: "moment", TargetID: "moment_feed_1"}); err != nil {
		t.Fatalf("dismiss built-in moment failed: %v", err)
	}
}

func TestInMemoryRepositoryDismissNotFoundForUnknownType(t *testing.T) {
	repo := NewInMemoryRepository()

	if err := repo.Dismiss(nil, dismissCommand{ViewerUserID: "u1", TargetType: "unknown_type", TargetID: "x"}); err == nil {
		t.Fatal("expected error for unknown type")
	}
}
