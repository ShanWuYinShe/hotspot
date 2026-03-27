package search

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

type stubRepository struct {
	response SearchSuggestResponse
	err      error
}

func (r stubRepository) Suggest(context.Context, SearchSuggestRequest) (SearchSuggestResponse, error) {
	return r.response, r.err
}

func TestServiceSuggestDelegatesToRepository(t *testing.T) {
	expected := SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "hot"}}}}
	service := NewServiceWithRepo(stubRepository{response: expected})

	result, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Type != "tag" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestServiceSuggestReturnsRepositoryError(t *testing.T) {
	service := NewServiceWithRepo(stubRepository{err: errors.New("boom")})

	_, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceSuggestAppliesResponseDefaultLimit(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.ResponseDefaults.SearchSuggestCount = 1
		return cfg
	})
	service := NewServiceWithSource(stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "hot"}}, {Type: "user", Payload: map[string]any{"user_id": "u1"}}}}}, source)
	result, err := service.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("unexpected limited result: %#v", result)
	}
}
