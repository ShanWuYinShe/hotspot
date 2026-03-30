package feed

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type stubRepository struct {
	momentsResult  PaginatedMoments
	groupsResult   PaginatedGroups
	usersResult    PaginatedUsers
	trendingResult TrendingBundle
	err            error
	lastMoments    momentsQuery
	lastGroups     groupsQuery
	lastUsers      usersQuery
	lastViewerID   string
}

func (r *stubRepository) ListMoments(_ context.Context, query momentsQuery) (PaginatedMoments, error) {
	r.lastMoments = query
	return r.momentsResult, r.err
}

func (r *stubRepository) ListGroups(_ context.Context, query groupsQuery) (PaginatedGroups, error) {
	r.lastGroups = query
	return r.groupsResult, r.err
}

func (r *stubRepository) ListUsers(_ context.Context, query usersQuery) (PaginatedUsers, error) {
	r.lastUsers = query
	return r.usersResult, r.err
}

func (r *stubRepository) ListTrending(_ context.Context, viewerUserID string) (TrendingBundle, error) {
	r.lastViewerID = viewerUserID
	return r.trendingResult, r.err
}

func TestServiceListMomentsUsesPersistenceRepositoryWhenRepoMissing(t *testing.T) {
	service := NewServiceWithSource(nil, appconfig.NewStaticSource(config.Config{}))
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.ListMoments(ctx, ListFeedMomentsRequest{Source: "following", ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestServiceListMomentsAppliesDefaultsAndPagination(t *testing.T) {
	repo := &stubRepository{}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.ListMoments(ctx, ListFeedMomentsRequest{Source: "nearby", ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}, Pagination: &PaginationParams{Limit: 25, Cursor: "abc"}})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if repo.lastMoments.Source != "nearby" || repo.lastMoments.Radius != 5000 || repo.lastMoments.Limit != 25 || repo.lastMoments.Cursor != "abc" {
		t.Fatalf("unexpected query: %#v", repo.lastMoments)
	}
}

func TestServiceListMomentsClampsRadiusAndLimit(t *testing.T) {
	repo := &stubRepository{}
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Limits.SearchRadiusMax = 1200
		return cfg
	})
	service := NewServiceWithSource(repo, source)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})
	radius := 9999

	_, err := service.ListMoments(ctx, ListFeedMomentsRequest{Source: "all", Radius: &radius, ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}, Pagination: &PaginationParams{Limit: 999}})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if repo.lastMoments.Radius != 1200 || repo.lastMoments.Limit != 100 {
		t.Fatalf("unexpected normalized query: %#v", repo.lastMoments)
	}
}

func TestServiceListMomentsRejectsInvalidSource(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.ListMoments(ctx, ListFeedMomentsRequest{Source: "bad", ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, ErrInvalidFeedRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceListUsersRejectsMissingReferencePoint(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.ListUsers(ctx, ListFeedUsersRequest{})
	if !errors.Is(err, ErrInvalidFeedRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceListGroupsRequiresPrincipal(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})

	_, err := service.ListGroups(context.Background(), ListFeedGroupsRequest{ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, ErrInvalidFeedRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceListTrendingAppliesResponseDefaults(t *testing.T) {
	repo := &stubRepository{trendingResult: TrendingBundle{
		TrendingTags: []TrendingTagItem{{Name: "hot"}, {Name: "new"}},
		HotMoments:   []MomentItem{{MomentID: "m1"}, {MomentID: "m2"}},
	}}
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.TrendingTagsCount = 1
		cfg.ResponseDefaults.HotMomentsCount = 1
		return cfg
	})
	service := NewServiceWithSource(repo, source)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	result, err := service.ListTrending(ctx)
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(result.TrendingTags) != 1 || len(result.HotMoments) != 1 || repo.lastViewerID != "u1" {
		t.Fatalf("unexpected result: %#v viewer=%q", result, repo.lastViewerID)
	}
}

func TestServiceListTrendingReturnsRepositoryError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{err: errors.New("boom")})

	_, err := service.ListTrending(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
