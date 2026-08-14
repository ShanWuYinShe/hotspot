package feed

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
)

type stubRepository struct {
	momentsResult        PaginatedMoments
	groupsResult         PaginatedGroups
	usersResult          PaginatedUsers
	trendingResult       TrendingResult
	err                  error
	lastFollowingMoments momentsQuery
	lastNearbyMoments    momentsQuery
	lastGroups           groupsQuery
	lastUsers            usersQuery
	lastViewerID         string
}

func (r *stubRepository) ListFollowingMoments(_ context.Context, query momentsQuery) (PaginatedMoments, error) {
	r.lastFollowingMoments = query
	return r.momentsResult, r.err
}

func (r *stubRepository) ListNearbyMoments(_ context.Context, query momentsQuery) (PaginatedMoments, error) {
	r.lastNearbyMoments = query
	return r.momentsResult, r.err
}

func (r *stubRepository) ListNearbyGroups(_ context.Context, query groupsQuery) (PaginatedGroups, error) {
	r.lastGroups = query
	return r.groupsResult, r.err
}

func (r *stubRepository) ListNearbyUsers(_ context.Context, query usersQuery) (PaginatedUsers, error) {
	r.lastUsers = query
	return r.usersResult, r.err
}

func (r *stubRepository) ListTrending(_ context.Context, viewerUserID string) (TrendingResult, error) {
	r.lastViewerID = viewerUserID
	return r.trendingResult, r.err
}

func (r *stubRepository) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}

func TestServiceFollowingAppliesDefaultsAndPagination(t *testing.T) {
	repo := &stubRepository{}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.Following(ctx, GetFeedFollowingRequest{ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}, Pagination: &PaginationParams{Limit: 25, Cursor: "abc"}})
	if err != nil {
		t.Fatalf("Following returned error: %v", err)
	}
	if repo.lastFollowingMoments.Limit != 25 || repo.lastFollowingMoments.Cursor != "abc" {
		t.Fatalf("unexpected query: %#v", repo.lastFollowingMoments)
	}
}

func TestServiceNearbyClampsRadiusAndLimit(t *testing.T) {
	repo := &stubRepository{}
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Limits.SearchRadiusMax = 1200
		return cfg
	})
	service := NewServiceWithSource(repo, source)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})
	radius := 9999

	_, err := service.Nearby(ctx, NearbyMomentsRequest{Radius: &radius, ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}, Pagination: &PaginationParams{Limit: 999}})
	if err != nil {
		t.Fatalf("Nearby returned error: %v", err)
	}
	if repo.lastNearbyMoments.Radius != 1200 || repo.lastNearbyMoments.Limit != 100 {
		t.Fatalf("unexpected normalized query: %#v", repo.lastNearbyMoments)
	}
}

func TestServiceNearbyFallsBackWhenMissingReferencePoint(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	// reference_point 可选:缺失时按回退链解析(用户位置 → default_location),不再报错
	_, err := service.Nearby(ctx, NearbyMomentsRequest{Radius: nil, ReferencePoint: nil, Pagination: nil})
	if err != nil {
		t.Fatalf("missing reference_point should fall back to default, got %v", err)
	}
	if repo := service.repo.(*stubRepository); repo.lastNearbyMoments.ReferencePoint == (GeoPoint{}) {
		t.Fatalf("expected a resolved reference_point in query, got zero value")
	}
}

func TestServiceNearbyRejectsInvalidReferencePoint(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.Nearby(ctx, NearbyMomentsRequest{ReferencePoint: &GeoPoint{Latitude: 999, Longitude: 2}})
	if !errors.Is(err, ErrInvalidFeedRequest) {
		t.Fatalf("expected invalid request for invalid reference_point, got %v", err)
	}
}

func TestServiceNearbyUsersFallsBackToDefaultLocation(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	_, err := service.NearbyUsers(ctx, GetFeedNearbyUsersRequest{})
	if err == ErrInvalidFeedRequest {
		t.Fatalf("missing reference_point should fall back to default, got %v", err)
	}
}

func TestServiceNearbyGroupsWithOptionalAuth(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})

	_, err := service.NearbyGroups(context.Background(), GetFeedNearbyGroupsRequest{ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}})
	if err != nil {
		t.Fatalf("NearbyGroups with optional auth returned error: %v", err)
	}
}

func TestServiceTrendingAppliesResponseDefaults(t *testing.T) {
	repo := &stubRepository{trendingResult: TrendingResult{
		HotMoments: []MomentItem{{MomentID: "m1"}, {MomentID: "m2"}},
	}}
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.HotMomentsCount = 1
		return cfg
	})
	service := NewServiceWithSource(repo, source)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "u1"})

	result, err := service.Trending(ctx)
	if err != nil {
		t.Fatalf("Trending returned error: %v", err)
	}
	if len(result.HotMoments) != 1 || repo.lastViewerID != "u1" {
		t.Fatalf("unexpected result: %#v viewer=%q", result, repo.lastViewerID)
	}
}

func TestServiceTrendingReturnsRepositoryError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{err: errors.New("boom")})

	_, err := service.Trending(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
