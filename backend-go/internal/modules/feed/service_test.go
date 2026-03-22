package feed

import (
	"context"
	"errors"
	"testing"
)

type stubRepository struct {
	listMomentsResult  PaginatedFeedItems
	listGroupsResult   PaginatedFeedItems
	listUsersResult    PaginatedFeedItems
	listTrendingResult TrendingBundle
	err                error
}

func (r stubRepository) ListMoments(context.Context) (PaginatedFeedItems, error) {
	return r.listMomentsResult, r.err
}

func (r stubRepository) ListGroups(context.Context) (PaginatedFeedItems, error) {
	return r.listGroupsResult, r.err
}

func (r stubRepository) ListUsers(context.Context) (PaginatedFeedItems, error) {
	return r.listUsersResult, r.err
}

func (r stubRepository) ListTrending(context.Context) (TrendingBundle, error) {
	return r.listTrendingResult, r.err
}

func TestServiceListMomentsDelegatesToRepository(t *testing.T) {
	expected := PaginatedFeedItems{Items: []map[string]any{{"moment_id": "moment_test_1"}}}
	service := NewServiceWithRepo(stubRepository{listMomentsResult: expected})

	result, err := service.ListMoments(context.Background())
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0]["moment_id"] != "moment_test_1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestServiceListTrendingReturnsRepositoryError(t *testing.T) {
	service := NewServiceWithRepo(stubRepository{err: errors.New("boom")})

	_, err := service.ListTrending(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
