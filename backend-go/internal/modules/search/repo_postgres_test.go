package search

import (
	"context"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type searchRowValues struct {
	values []any
	err    error
}

func (r searchRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *float64:
			*d = r.values[i].(float64)
		case *int:
			*d = r.values[i].(int)
		}
	}
	return nil
}

type searchRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *searchRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *searchRowsValues) Scan(dest ...any) error {
	row := searchRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *searchRowsValues) Close() error { return nil }
func (r *searchRowsValues) Err() error   { return r.err }

type searchBackendStub struct {
	queryRowsByQuery map[string][][]any
	queryErrs        map[string]error
}

func (b *searchBackendStub) Ping(context.Context) error { return nil }
func (b *searchBackendStub) Close() error               { return nil }
func (b *searchBackendStub) Exec(context.Context, string, ...any) error {
	return nil
}
func (b *searchBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return searchRowValues{err: persistence.ErrNotImplemented}
}
func (b *searchBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if rows, ok := b.queryRowsByQuery[query]; ok {
		return &searchRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

type fallbackSearchRepo struct {
	used bool
}

func (r *fallbackSearchRepo) Suggest(context.Context, SearchSuggestRequest) (SearchSuggestResponse, error) {
	r.used = true
	return SearchSuggestResponse{Items: []SearchSuggestItem{{Type: "tag", Payload: map[string]any{"name": "fallback"}}}}, nil
}

func newSearchFactory(backend persistence.PostgresQueryBackend) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresRepositoryFallsBackWithoutDatabase(t *testing.T) {
	fallback := &fallbackSearchRepo{}
	repo := NewPostgresRepository(nil, fallback)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if !fallback.used || len(resp.Items) != 1 || resp.Items[0].Payload["name"] != "fallback" {
		t.Fatalf("expected fallback response, got %#v used=%v", resp, fallback.used)
	}
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabaseOrFallback(t *testing.T) {
	repo := NewPostgresRepository(nil, nil)

	_, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryReturnsEmptyForBlankPrefix(t *testing.T) {
	repo := NewPostgresRepository(newSearchFactory(&searchBackendStub{}), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "   "})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty items, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryFiltersUnknownTypesAsAll(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestTags:    {{"hot", 8}},
		querySearchSuggestUsers:   {{"user_1", "hot-user", "", "", "南天门", 35.0, 112.0}},
		querySearchSuggestGroups:  {{"group_1", "hot-group", "", "", "南天门", 35.0, 112.0, 5}},
		querySearchSuggestMoments: {{"moment_1", "user_1", "Alice", "hot moment", "南天门", 35.0, 112.0, 9}},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"weird"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 4 {
		t.Fatalf("expected all types, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryAppliesTypeFilter(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {{"user_1", "hot-user", "", "", "南天门", 35.0, 112.0}},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Type != "user" {
		t.Fatalf("expected only user result, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryHonorsNicknameSearchableOnlyForNickname(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {
			{"user_bio", "plain-user", "", "hot bio", "南天门", 35.0, 112.0},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Payload["user_id"] != "user_bio" {
		t.Fatalf("expected bio-matched user result, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryRanksExactBeforePrefixBeforeContains(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestTags: {
			{"xxhot", 10},
			{"hotzone", 8},
			{"hot", 5},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %#v", resp.Items)
	}
	if resp.Items[0].Payload["name"] != "hot" || resp.Items[1].Payload["name"] != "hotzone" || resp.Items[2].Payload["name"] != "xxhot" {
		t.Fatalf("unexpected order: %#v", resp.Items)
	}
}

func TestPostgresRepositoryUsesDistanceBeforePopularityWhenMatchRankTies(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestGroups: {
			{"group_far", "hot-group", "", "", "远处", 35.1, 112.0, 100},
			{"group_near", "hot-group", "", "", "近处", 35.0001, 112.0, 1},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"group"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].Payload["group_id"] != "group_near" {
		t.Fatalf("expected nearer group first, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryAppliesRadiusFilter(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {
			{"user_near", "hot-near", "", "", "近处", 35.0001, 112.0},
			{"user_far", "hot-far", "", "", "远处", 35.1, 112.0},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}, Radius: 2000})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Payload["user_id"] != "user_near" {
		t.Fatalf("expected radius filtered result, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryDedupesSameTypeSameID(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestTags: {
			{"hot", 3},
			{"hot", 2},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected deduped items, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryReturnsRepositoryError(t *testing.T) {
	backend := &searchBackendStub{queryErrs: map[string]error{querySearchSuggestTags: persistence.ErrNotImplemented}}
	repo := NewPostgresRepository(newSearchFactory(backend), nil)

	_, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPostgresRepositoryTagQueryRequiresPublicMoments(t *testing.T) {
	if !strings.Contains(querySearchSuggestTags, "JOIN moments m ON m.moment_id = mt.moment_id") {
		t.Fatalf("expected tag query to join moments: %s", querySearchSuggestTags)
	}
	if !strings.Contains(querySearchSuggestTags, "m.privacy_level = 'public'") {
		t.Fatalf("expected tag query to filter public moments: %s", querySearchSuggestTags)
	}
}

func TestPostgresRepositoryUserQueryRequiresPublicProfiles(t *testing.T) {
	if !strings.Contains(querySearchSuggestUsers, "us.privacy_level = 'public'") {
		t.Fatalf("expected user query to filter public profiles: %s", querySearchSuggestUsers)
	}
}

func TestPostgresRepositoryGroupQueryExcludesDeletedGroups(t *testing.T) {
	if !strings.Contains(querySearchSuggestGroups, "g.deleted_at IS NULL") {
		t.Fatalf("expected group query to exclude deleted groups: %s", querySearchSuggestGroups)
	}
}

func TestPostgresRepositoryMomentQueryRequiresPublicMoments(t *testing.T) {
	if !strings.Contains(querySearchSuggestMoments, "m.privacy_level = 'public'") {
		t.Fatalf("expected moment query to filter public moments: %s", querySearchSuggestMoments)
	}
}
