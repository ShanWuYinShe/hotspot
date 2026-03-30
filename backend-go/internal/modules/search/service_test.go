package search

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type stubRepository struct {
	response SearchSuggestResponse
	err      error
	lastReq  SearchSuggestRequest
}

func (r *stubRepository) Suggest(_ context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	r.lastReq = req
	return r.response, r.err
}

func TestServiceSuggestDelegatesToRepository(t *testing.T) {
	repo := &stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "hot"}}}}}
	service := NewServiceWithRepo(repo)

	result, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}, Radius: 150})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Type != "tag" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if repo.lastReq.Radius != 150 {
		t.Fatalf("expected radius passthrough, got %#v", repo.lastReq)
	}
}

func TestServiceSuggestReturnsRepositoryError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepository{err: errors.New("boom")})

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceSuggestUsesPersistenceRepositoryWhenRepoMissing(t *testing.T) {
	service := NewServiceWithSource(nil, appconfig.NewStaticSource(config.Config{}))

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestServiceSuggestAppliesDefaultRadius(t *testing.T) {
	repo := &stubRepository{}
	service := NewServiceWithRepo(repo)

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if repo.lastReq.Radius != defaultSearchSuggestRadius {
		t.Fatalf("expected default radius %d, got %d", defaultSearchSuggestRadius, repo.lastReq.Radius)
	}
}

func TestServiceSuggestClampsRadiusToAppConfigLimit(t *testing.T) {
	repo := &stubRepository{}
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Limits.SearchRadiusMax = 1200
		return cfg
	})
	service := NewServiceWithSource(repo, source)

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}, Radius: 5000})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if repo.lastReq.Radius != 1200 {
		t.Fatalf("expected clamped radius 1200, got %d", repo.lastReq.Radius)
	}
}

func TestServiceSuggestClampsRadiusToDefaultSpecMax(t *testing.T) {
	repo := &stubRepository{}
	service := NewServiceWithRepo(repo)

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}, Radius: 50001})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if repo.lastReq.Radius != maxSearchSuggestRadius {
		t.Fatalf("expected service to clamp radius to default spec max %d, got %d", maxSearchSuggestRadius, repo.lastReq.Radius)
	}
}

func TestServiceSuggestAppliesResponseDefaultLimit(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.SearchSuggestCount = 1
		return cfg
	})
	service := NewServiceWithSource(&stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "hot"}}, {Type: "user", Payload: map[string]any{"user_id": "u1"}}}}}, source)
	result, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("unexpected limited result: %#v", result)
	}
}

func TestServiceSuggestPreservesTypeSelection(t *testing.T) {
	repo := &stubRepository{}
	service := NewServiceWithRepo(repo)

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", ReferencePoint: GeoPoint{Latitude: 1, Longitude: 2}, Types: []string{"user", "group"}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(repo.lastReq.Types) != 2 || repo.lastReq.Types[0] != "user" || repo.lastReq.Types[1] != "group" {
		t.Fatalf("expected types passthrough, got %#v", repo.lastReq.Types)
	}
}

func TestIsValidGeoPoint(t *testing.T) {
	if !isValidGeoPoint(GeoPoint{Latitude: 35, Longitude: 112}) {
		t.Fatal("expected valid geo point")
	}
	if isValidGeoPoint(GeoPoint{Latitude: 91, Longitude: 112}) {
		t.Fatal("expected invalid latitude")
	}
	if isValidGeoPoint(GeoPoint{Latitude: 35, Longitude: 181}) {
		t.Fatal("expected invalid longitude")
	}
}
