package search

import (
	"context"
	"hotspot/backend-go/internal/platform/testdb"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type searchBackendStub struct {
	queryRowsByQuery map[string][][]any
	queryErrs        map[string]error
	// lastQuery / lastArgs capture the most recent Query call for tests that
	// need to inspect what was sent to the backend.
	lastQuery string
	lastArgs  []any
}

func (b *searchBackendStub) Ping(context.Context) error { return nil }
func (b *searchBackendStub) Close() error               { return nil }
func (b *searchBackendStub) Exec(context.Context, string, ...any) error {
	return nil
}
func (b *searchBackendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *searchBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	b.lastQuery = query
	b.lastArgs = args
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if rows, ok := b.queryRowsByQuery[query]; ok {
		return &testdb.RowsValues{Rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func newSearchFactory(backend persistence.PostgresQueryBackend) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabase(t *testing.T) {
	repo := NewPostgresRepository(nil)

	_, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryReturnsEmptyForBlankPrefix(t *testing.T) {
	repo := NewPostgresRepository(newSearchFactory(&searchBackendStub{}))

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
		querySearchSuggestUsers:   {{"user_1", "hot-user", "", "", "南天门", 35.0, 112.0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0, true, true, true}},
		querySearchSuggestGroups:  {{"group_1", "hot-group", "", "", "南天门", 35.0, 112.0, 5, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 2}},
		querySearchSuggestMoments: {{"moment_1", "user_1", "Alice", "", "hot moment", "南天门", 35.0, 112.0, "", 9}},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"weird"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 4 {
		t.Fatalf("expected all types, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryAppliesTypeFilter(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {{"user_1", "hot-user", "", "", "南天门", 35.0, 112.0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0, true, true, true}},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
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
			{"user_bio", "plain-user", "", "hot bio", "南天门", 35.0, 112.0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0, true, true, true},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Payload["user"].(map[string]any)["user_id"] != "user_bio" {
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
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
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
			{"group_far", "hot-group", "", "", "远处", 35.1, 112.0, 100, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 2},
			{"group_near", "hot-group", "", "", "近处", 35.0001, 112.0, 1, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 2},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"group"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].Payload["group"].(map[string]any)["group_id"] != "group_near" {
		t.Fatalf("expected nearer group first, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryAppliesRadiusFilter(t *testing.T) {
	// Radius filtering happens in Go: distance is recomputed from the row's
	// latitude/longitude and rows farther than the radius are dropped. The mock
	// backend returns both rows; the far one must be filtered out by the repo.
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {
			{"user_near", "hot-near", "", "", "近处", 35.0001, 112.0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0, true, true, true},
			{"user_far", "hot-far", "", "", "远处", 35.1, 112.0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0, true, true, true},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}, Radius: 2000})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected only the near user after Go-side radius filter, got %#v", resp.Items)
	}
	if resp.Items[0].Payload["user"].(map[string]any)["user_id"] != "user_near" {
		t.Fatalf("expected user_near to survive the filter, got %#v", resp.Items[0].Payload)
	}
}

func TestPostgresRepositoryDedupesSameTypeSameID(t *testing.T) {
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestTags: {
			{"hot", 3},
			{"hot", 2},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected deduped items, got %#v", resp.Items)
	}
}

func TestPostgresRepositoryReturnsRepositoryError(t *testing.T) {
	backend := &searchBackendStub{queryErrs: map[string]error{querySearchSuggestTags: persistence.ErrNotImplemented}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	_, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"tag"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
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

func TestPostgresRepositoryNullsFieldsHiddenByVisibilitySwitches(t *testing.T) {
	// L1 可见性门控（api_spec_authz.md #28/#108）+ 类型稳定性（api_spec_base.md）：
	// location_visible/presence_visible/stats_visible=false 的字段以显式 null
	// 返回（键恒在、值为 null）——精确坐标 + distance/bearing 仍绝不泄漏。
	backend := &searchBackendStub{queryRowsByQuery: map[string][][]any{
		querySearchSuggestUsers: {
			{"user_hidden", "hot-hidden", "", "", "某处", 35.0, 112.0, "online", time.Now(), []string{"tag1"}, 3, 2, 1, 1, 1, false, false, false},
			{"user_open", "hot-open", "", "", "某处", 35.0, 112.0, "online", time.Now(), []string{}, 3, 2, 1, 1, 1, true, true, true},
		},
	}}
	repo := NewPostgresRepository(newSearchFactory(backend))

	resp, err := repo.Suggest(context.Background(), SearchSuggestRequest{Prefix: "hot", Types: []string{"user"}, ReferencePoint: &GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	byID := map[string]map[string]any{}
	for _, item := range resp.Items {
		byID[item.Payload["user"].(map[string]any)["user_id"].(string)] = item.Payload
	}
	hidden, ok := byID["user_hidden"]
	if !ok {
		t.Fatalf("expected hidden user to remain searchable, got %#v", resp.Items)
	}
	for _, key := range []string{"location", "presence_status", "last_online_at", "stats"} {
		val, present := hidden[key]
		if !present {
			t.Fatalf("expected %q key to be present (explicit null) for hidden user", key)
		}
		if val != nil {
			t.Fatalf("expected %q to be null for hidden user, got %#v", key, val)
		}
	}
	open, ok := byID["user_open"]
	if !ok {
		t.Fatalf("expected open user in results, got %#v", resp.Items)
	}
	for _, key := range []string{"location", "presence_status", "last_online_at", "stats"} {
		if _, present := open[key]; !present {
			t.Fatalf("expected %q to be present for open user, got %#v", key, open)
		}
	}
}
