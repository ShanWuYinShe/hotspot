package groups

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"reflect"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type groupBackendStub struct {
	rows      [][]any
	queryErrs map[string]error
	queryRow  persistence.Row
	execErrs  map[string]error
}

func (b *groupBackendStub) Ping(context.Context) error { return nil }
func (b *groupBackendStub) Close() error               { return nil }
func (b *groupBackendStub) Exec(_ context.Context, query string, _ ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	return nil
}
func (b *groupBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if query == queryGetGroup {
		return b.queryRow
	}
	if query == queryCountGroupMembers {
		return testdb.RowValues{Values: []any{1}}
	}
	if query == queryGetGroupMembership {
		return testdb.RowValues{Err: pgx.ErrNoRows}
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *groupBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if strings.Contains(query, "FROM groups g") {
		if err := b.queryErrs[querySearchGroups]; err != nil {
			return nil, err
		}
		return &testdb.RowsValues{Rows: b.rows}, nil
	}
	if query == querySearchGroups {
		return &testdb.RowsValues{Rows: b.rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryCreateGetAndSearch(t *testing.T) {
	backend := &groupBackendStub{
		// 列序对齐 queryGetGroup/querySearchGroups：... is_searchable, members_searchable, max_members, ...
		rows:      [][]any{{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_alice", nil, nil, []string{"hiking"}, 1, 1}},
		queryErrs: map[string]error{},
		queryRow:  testdb.RowValues{Values: []any{"group_1", "Hot Group", "", "desc", false, "南天门", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_alice", nil, nil, []string{"hiking"}, 1}},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	created, err := repo.Create(context.Background(), "user_alice", CreateGroupRequest{Name: "Hot Group", Location: Location{LocationName: "南天门"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Group.GroupID == "" {
		t.Fatalf("expected created group id, got %#v", created)
	}
	if (created.OwnerID == nil || *created.OwnerID != "user_alice") || created.CreatedAt == nil {
		t.Fatalf("expected owner metadata on created group, got %#v", created)
	}
	item, ok, err := repo.Get(context.Background(), "group_1")
	if err != nil || !ok || item.Group.GroupID != "group_1" {
		t.Fatalf("unexpected item: %#v err=%v", item, err)
	}
	if item.OwnerID == nil || *item.OwnerID != "user_alice" {
		t.Fatalf("expected owner id in fetched item, got %#v", item)
	}
	list, err := repo.Search(context.Background(), SearchGroupsRequest{Keyword: "Hot Group", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil || len(list.Items) != 1 || list.Items[0].Group.GroupID != "group_1" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryGetReturnsNotFoundWithoutFallback(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{}, queryRow: testdb.RowValues{Err: errors.New("no rows")}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, ok, err := repo.Get(context.Background(), "missing")
	if err == nil && ok {
		t.Fatalf("expected not found, got ok=%v err=%v", ok, err)
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{}, queryRow: testdb.RowValues{Err: persistence.ErrNotImplemented}, execErrs: map[string]error{queryInsertGroup: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Create(context.Background(), "user_alice", CreateGroupRequest{Name: "Hot Group", Location: Location{LocationName: "南天门"}})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositorySearchReturnsErrorOnQueryFailure(t *testing.T) {
	backend := &groupBackendStub{rows: nil, queryErrs: map[string]error{querySearchGroups: errors.New("boom")}, queryRow: testdb.RowValues{Err: persistence.ErrNotImplemented}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Search(context.Background(), SearchGroupsRequest{Keyword: "Hot Group", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err == nil {
		t.Fatal("expected search error")
	}
}

// TestPostgresRepositorySearchFiltersSortsAndPaginatesInGo pins the Go-side
// search pipeline: exact radius filtering via the geo plan, distance ordering
// and keyset pagination over the full sorted result list (the SQL no longer
// computes distances or applies cursor predicates).
func TestPostgresRepositorySearchFiltersSortsAndPaginatesInGo(t *testing.T) {
	backend := &groupBackendStub{
		rows: [][]any{
			{"g_1", "Alpha One", "", "", false, "loc_a", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_a", nil, nil, []string{}, 3, 1},
			{"g_2", "Beta Bar", "", "alpha den", false, "loc_b", 35.0005, 112.0, "direct", false, nil, true, true, 200, "user_b", nil, nil, []string{}, 1, 1},
			{"g_3", "Alpha Three", "", "", false, "loc_c", 35.01, 112.0, "direct", false, nil, true, true, 200, "user_c", nil, nil, []string{}, 2, 1},
		},
		queryErrs: map[string]error{},
		queryRow:  testdb.RowValues{Err: persistence.ErrNotImplemented},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	radius := 100
	newReq := func(cursor string) SearchGroupsRequest {
		return SearchGroupsRequest{
			Keyword:        "alpha",
			Radius:         &radius,
			SortBy:         searchSortDistance,
			ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0},
			Pagination:     &PaginationParams{Limit: 1, Cursor: cursor},
		}
	}

	page1, err := repo.Search(context.Background(), newReq(""))
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if len(page1.Items) != 1 || page1.Items[0].Group.GroupID != "g_1" {
		t.Fatalf("expected g_1 first (nearest), got %#v", page1.Items)
	}
	if page1.Items[0].Location.Distance == nil || *page1.Items[0].Location.Distance != 0 {
		t.Fatalf("expected zero distance on reference-point match, got %#v", page1.Items[0].Location)
	}
	if page1.NextCursor == nil || *page1.NextCursor == "" {
		t.Fatal("expected next cursor on page 1")
	}

	page2, err := repo.Search(context.Background(), newReq(*page1.NextCursor))
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].Group.GroupID != "g_2" {
		t.Fatalf("expected g_2 second (farther, still within radius), got %#v", page2.Items)
	}
	if page2.NextCursor != nil && *page2.NextCursor != "" {
		t.Fatalf("expected no next cursor on page 2, got %q", *page2.NextCursor)
	}

	// g_3 sits ~1.1km away — outside the 100m radius — and must be dropped by
	// the Go radius filter on every page.
	for _, page := range []PaginatedGroups{page1, page2} {
		for _, item := range page.Items {
			if item.Group.GroupID == "g_3" {
				t.Fatalf("g_3 should have been excluded by the Go radius filter, got %#v", page.Items)
			}
		}
	}
}

// TestPostgresRepositorySearchGoSorts pins the Go-side ordering for each sort
// mode: relevance (nameMatch score then members), popularity, newest and
// distance (tie-broken by group_id).
func TestPostgresRepositorySearchGoSorts(t *testing.T) {
	createdAt := func(year int) time.Time {
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	backend := &groupBackendStub{
		rows: [][]any{
			{"g_a", "Alpha Hub", "", "", false, "loc", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_a", createdAt(2024), nil, []string{}, 10, 1},
			{"g_b", "Beacon", "", "alpha outpost", false, "loc", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_b", createdAt(2026), nil, []string{}, 20, 1},
			{"g_c", "Zulu Alpha", "", "", false, "loc", 35.0, 112.0, "direct", false, nil, true, true, 200, "user_c", createdAt(2025), nil, []string{}, 30, 1},
		},
		queryErrs: map[string]error{},
		queryRow:  testdb.RowValues{Err: persistence.ErrNotImplemented},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	radius := 5000
	newReq := func(sortBy string) SearchGroupsRequest {
		return SearchGroupsRequest{Keyword: "alpha", Radius: &radius, SortBy: sortBy, ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}}
	}

	// relevance: name contains alpha (score 2) beats description-only (score 1);
	// equal scores tie-break by members_count DESC.
	relevance, err := repo.Search(context.Background(), newReq(searchSortRelevance))
	if err != nil {
		t.Fatalf("relevance search: %v", err)
	}
	if got := groupSearchIDs(relevance.Items); !reflect.DeepEqual(got, []string{"g_c", "g_a", "g_b"}) {
		t.Fatalf("relevance order = %v, want [g_c g_a g_b]", got)
	}

	// popular: members_count DESC.
	popular, err := repo.Search(context.Background(), newReq(searchSortPopular))
	if err != nil {
		t.Fatalf("popular search: %v", err)
	}
	if got := groupSearchIDs(popular.Items); !reflect.DeepEqual(got, []string{"g_c", "g_b", "g_a"}) {
		t.Fatalf("popular order = %v, want [g_c g_b g_a]", got)
	}

	// newest: created_at DESC.
	newest, err := repo.Search(context.Background(), newReq(searchSortNewest))
	if err != nil {
		t.Fatalf("newest search: %v", err)
	}
	if got := groupSearchIDs(newest.Items); !reflect.DeepEqual(got, []string{"g_b", "g_c", "g_a"}) {
		t.Fatalf("newest order = %v, want [g_b g_c g_a]", got)
	}

	// distance: all rows sit on the reference point, so ties resolve by
	// group_id ASC.
	distance, err := repo.Search(context.Background(), newReq(searchSortDistance))
	if err != nil {
		t.Fatalf("distance search: %v", err)
	}
	if got := groupSearchIDs(distance.Items); !reflect.DeepEqual(got, []string{"g_a", "g_b", "g_c"}) {
		t.Fatalf("distance order = %v, want [g_a g_b g_c]", got)
	}
}

func groupSearchIDs(items []GroupItem) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.Group.GroupID
	}
	return ids
}
