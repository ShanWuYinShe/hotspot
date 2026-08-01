package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
)

// feedReactionAggregator records AggregateMoments calls and returns canned
// per-moment reaction aggregates.
type feedReactionAggregator struct {
	calls   [][]string
	viewers []string
	result  map[string][]Reaction
	err     error
}

func (a *feedReactionAggregator) AggregateMoments(_ context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error) {
	a.calls = append(a.calls, append([]string(nil), momentIDs...))
	a.viewers = append(a.viewers, viewerID)
	if a.err != nil {
		return nil, a.err
	}
	return a.result, nil
}

// feedLocationProvider serves the reference_point fallback level 2.
type feedLocationProvider struct {
	point geo.Point
	found bool
}

func (p feedLocationProvider) RecentLocation(_ context.Context, _ string) (geo.Point, bool) {
	return p.point, p.found
}

// feedImageRepo overrides FetchMomentImageRows of the shared stubRepository.
type feedImageRepo struct {
	stubRepository
	rows map[string][]momentImageRow
	err  error
}

func (r *feedImageRepo) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.rows, nil
}

// feedImageResolver presigns storage keys deterministically.
type feedImageResolver struct {
	presign map[string]string
}

func (r feedImageResolver) ResolveMomentImages(_ context.Context, _ []string, _ string) (map[string]ImageAsset, error) {
	return map[string]ImageAsset{}, nil
}

func (r feedImageResolver) PresignGet(_ context.Context, storageKey string, _ time.Duration) (string, error) {
	if u, ok := r.presign[storageKey]; ok {
		return u, nil
	}
	return "https://cdn.example/" + storageKey, nil
}

func feedCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

// --- Following/Nearby 返回带 reactions 聚合的 items ---

func TestServiceFollowingFillsReactions(t *testing.T) {
	repo := &stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{
		{MomentID: "m1"},
		{MomentID: "m2"},
	}}}
	agg := &feedReactionAggregator{result: map[string][]Reaction{
		"m1": {{Emoji: "👍", Count: 2, ReactedByMe: true}},
	}}
	service := NewServiceWithRepo(repo)
	service.SetReactionAggregator(agg)

	items, err := service.Following(feedCtx("u1"), GetFeedFollowingRequest{})
	if err != nil {
		t.Fatalf("Following: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items.Items))
	}
	if len(items.Items[0].Reactions) != 1 || items.Items[0].Reactions[0].Emoji != "👍" || !items.Items[0].Reactions[0].ReactedByMe {
		t.Fatalf("expected m1 reactions filled, got %+v", items.Items[0].Reactions)
	}
	if len(items.Items[1].Reactions) != 0 {
		t.Fatalf("m2 has no reactions in stub, got %+v", items.Items[1].Reactions)
	}
	// fillReactions 用 viewer 身份调用聚合器(reacted_by_me 语义)。
	if len(agg.calls) != 1 || len(agg.calls[0]) != 2 || agg.calls[0][0] != "m1" || agg.calls[0][1] != "m2" {
		t.Fatalf("unexpected aggregate call: %+v", agg.calls)
	}
	if len(agg.viewers) != 1 || agg.viewers[0] != "u1" {
		t.Fatalf("expected viewer u1, got %+v", agg.viewers)
	}
}

func TestServiceNearbyFillsReactions(t *testing.T) {
	repo := &stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}
	agg := &feedReactionAggregator{result: map[string][]Reaction{"m1": {{Emoji: "❤️", Count: 1}}}}
	service := NewServiceWithRepo(repo)
	service.SetReactionAggregator(agg)

	items, err := service.Nearby(feedCtx("u1"), GetFeedNearbyRequest{ReferencePoint: &GeoPoint{Latitude: 1, Longitude: 2}})
	if err != nil {
		t.Fatalf("Nearby: %v", err)
	}
	if len(items.Items) != 1 || len(items.Items[0].Reactions) != 1 || items.Items[0].Reactions[0].Emoji != "❤️" {
		t.Fatalf("expected reactions filled, got %+v", items.Items)
	}
	if len(agg.calls) != 1 || agg.calls[0][0] != "m1" || agg.viewers[0] != "u1" {
		t.Fatalf("unexpected aggregate call: %+v calls=%+v", agg.calls, agg.viewers)
	}
}

func TestServiceFillReactionsAggregatorErrorIsSwallowed(t *testing.T) {
	repo := &stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}
	agg := &feedReactionAggregator{err: errors.New("boom")}
	service := NewServiceWithRepo(repo)
	service.SetReactionAggregator(agg)

	items, err := service.Following(feedCtx("u1"), GetFeedFollowingRequest{})
	if err != nil {
		t.Fatalf("aggregator error must not fail Following: %v", err)
	}
	if len(items.Items[0].Reactions) != 0 {
		t.Fatalf("reactions must stay empty on aggregator error, got %+v", items.Items[0].Reactions)
	}
}

func TestServiceFollowingWithoutAggregatorLeavesItemsUntouched(t *testing.T) {
	repo := &stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}
	service := NewServiceWithRepo(repo) // reactions == nil

	items, err := service.Following(feedCtx("u1"), GetFeedFollowingRequest{})
	if err != nil {
		t.Fatalf("Following: %v", err)
	}
	if len(items.Items[0].Reactions) != 0 {
		t.Fatalf("expected no reactions without aggregator, got %+v", items.Items[0].Reactions)
	}
}

// --- hydrateImages ---

func TestServiceHydrateImagesFillsItems(t *testing.T) {
	repo := &feedImageRepo{
		stubRepository: stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{
			{MomentID: "m1"},
			{MomentID: "m2"},
		}}},
		rows: map[string][]momentImageRow{
			"m1": {{MomentID: "m1", StorageKey: "sk/1", ThumbnailURL: "thumb/1", Width: 100, Height: 200, MimeType: "image/jpeg"}},
			"m2": {{MomentID: "m2", StorageKey: "sk/2"}},
		},
	}
	service := NewServiceWithSourceAndImages(repo, nil, feedImageResolver{})

	items, err := service.Following(feedCtx("u1"), GetFeedFollowingRequest{})
	if err != nil {
		t.Fatalf("Following: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items.Items))
	}
	if len(items.Items[0].Images) != 1 {
		t.Fatalf("expected 1 image for m1, got %+v", items.Items[0].Images)
	}
	img := items.Items[0].Images[0]
	if img.URL != "https://cdn.example/sk/1" || img.ThumbnailURL != "https://cdn.example/thumb/1" || img.Width != 100 || img.Height != 200 || img.MimeType != "image/jpeg" {
		t.Fatalf("unexpected hydrated image: %+v", img)
	}
	if len(items.Items[1].Images) != 1 {
		t.Fatalf("expected 1 image for m2 (thumbnail falls back to URL), got %+v", items.Items[1].Images)
	}
	if items.Items[1].Images[0].ThumbnailURL != "https://cdn.example/sk/2" {
		t.Fatalf("thumbnail must fall back to full URL, got %+v", items.Items[1].Images[0])
	}
}

func TestServiceHydrateImagesFetchErrorIsSwallowed(t *testing.T) {
	repo := &feedImageRepo{err: errors.New("boom")}
	service := NewServiceWithSourceAndImages(repo, nil, feedImageResolver{})

	items, err := service.Following(feedCtx("u1"), GetFeedFollowingRequest{})
	if err != nil {
		t.Fatalf("image fetch error must not fail Following: %v", err)
	}
	if len(items.Items) != 0 {
		t.Fatalf("expected empty items, got %+v", items.Items)
	}
}

// --- reference_point 回退链(显式 / 用户位置 / default_location) ---

func TestResolveReferencePointExplicitWins(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	service.SetLocationProvider(feedLocationProvider{point: geo.Point{Latitude: 3, Longitude: 4}, found: true})

	ref := service.resolveReferencePoint(feedCtx("u1"), &GeoPoint{Latitude: 1, Longitude: 2})
	if ref == nil || ref.Latitude != 1 || ref.Longitude != 2 {
		t.Fatalf("explicit reference_point must win, got %+v", ref)
	}
}

func TestResolveReferencePointUserRecentLocation(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})
	service.SetLocationProvider(feedLocationProvider{point: geo.Point{Latitude: 3.5, Longitude: 4.5}, found: true})

	ref := service.resolveReferencePoint(feedCtx("u1"), nil)
	if ref == nil || ref.Latitude != 3.5 || ref.Longitude != 4.5 {
		t.Fatalf("expected user recent location, got %+v", ref)
	}
}

func TestResolveReferencePointFallsBackToDefaultLocation(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Defaults.DefaultLocation.Point = appconfig.GeoPoint{Latitude: 5.5, Longitude: 6.5}
		return cfg
	})
	service := NewServiceWithSource(&stubRepository{}, source)

	ref := service.resolveReferencePoint(feedCtx("u1"), nil)
	if ref == nil || ref.Latitude != 5.5 || ref.Longitude != 6.5 {
		t.Fatalf("expected appConfig default_location, got %+v", ref)
	}
}

func TestResolveReferencePointFallsBackToGeoConstants(t *testing.T) {
	// 无 appConfig 注入 → 兜底到 geo 默认常量。
	service := NewServiceWithRepo(&stubRepository{})

	ref := service.resolveReferencePoint(feedCtx("u1"), nil)
	if ref == nil || ref.Latitude != geo.DefaultFallbackLatitude || ref.Longitude != geo.DefaultFallbackLongitude {
		t.Fatalf("expected geo default fallback, got %+v", ref)
	}
}

func TestResolveReferencePointInvalidExplicitReturnsNil(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{})

	ref := service.resolveReferencePoint(feedCtx("u1"), &GeoPoint{Latitude: 999, Longitude: 2})
	if ref != nil {
		t.Fatalf("invalid explicit point must yield nil, got %+v", ref)
	}
}
