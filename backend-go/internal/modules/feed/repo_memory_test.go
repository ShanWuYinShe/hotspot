package feed

import (
	"testing"
)

func TestInMemoryRepositoryListFollowingMomentsReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListFollowingMoments(nil, momentsQuery{})
	if err != nil {
		t.Fatalf("ListFollowingMoments returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(result.Items))
	}
}

func TestInMemoryRepositoryListNearbyMomentsReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListNearbyMoments(nil, momentsQuery{})
	if err != nil {
		t.Fatalf("ListNearbyMoments returned error: %v", err)
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
	if len(result.HotMoments) != 0 {
		t.Fatalf("expected empty hot moments, got %d", len(result.HotMoments))
	}
}

func TestInMemoryRepositoryListNearbyGroupsReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListNearbyGroups(nil, groupsQuery{})
	if err != nil {
		t.Fatalf("ListNearbyGroups returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty groups, got %d", len(result.Items))
	}
}

func TestInMemoryRepositoryListNearbyUsersReturnsEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.ListNearbyUsers(nil, usersQuery{})
	if err != nil {
		t.Fatalf("ListNearbyUsers returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty users, got %d", len(result.Items))
	}
}
