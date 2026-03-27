package feed

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
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

func TestServiceListTrendingAppliesResponseDefaultLimits(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.TrendingTagsCount = 1
		cfg.ResponseDefaults.HotMomentsCount = 1
		return cfg
	})
	service := NewServiceWithSource(stubRepository{listTrendingResult: TrendingBundle{
		TrendingTags: []map[string]any{{"name": "hot"}, {"name": "new"}},
		HotMoments:   []map[string]any{{"moment_id": "m1"}, {"moment_id": "m2"}},
	}}, source)
	result, err := service.ListTrending(context.Background())
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(result.TrendingTags) != 1 || len(result.HotMoments) != 1 {
		t.Fatalf("unexpected limited result: %#v", result)
	}
}
