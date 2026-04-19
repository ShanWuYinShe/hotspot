package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type feedRowValues struct {
	values []any
	err    error
}

func (r feedRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *float64:
			*d = r.values[i].(float64)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case *int:
			*d = r.values[i].(int)
		case *bool:
			*d = r.values[i].(bool)
		default:
			return errors.New("unsupported scan dest")
		}
	}
	return nil
}

type feedRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *feedRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *feedRowsValues) Scan(dest ...any) error {
	row := feedRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *feedRowsValues) Close() error { return nil }
func (r *feedRowsValues) Err() error   { return r.err }

type feedBackendStub struct {
	queryRowsByQuery map[string][][]any
	queryErrs        map[string]error
	queryRowByQuery  map[string]feedRowValues
	execErrs         map[string]error
	execCalls        []feedExecCall
}

type feedExecCall struct {
	query string
	args  []any
}

func (b *feedBackendStub) Ping(context.Context) error { return nil }
func (b *feedBackendStub) Close() error               { return nil }
func (b *feedBackendStub) Exec(_ context.Context, query string, args ...any) error {
	b.execCalls = append(b.execCalls, feedExecCall{query: query, args: args})
	if err := b.execErrs[query]; err != nil {
		return err
	}
	return nil
}
func (b *feedBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if row, ok := b.queryRowByQuery[query]; ok {
		return row
	}
	return feedRowValues{err: persistence.ErrNotImplemented}
}
func (b *feedBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if rows, ok := b.queryRowsByQuery[query]; ok {
		return &feedRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func newFeedFactory(backend persistence.PostgresQueryBackend) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabase(t *testing.T) {
	repo := NewPostgresRepository(nil)

	resp, err := repo.ListUsers(context.Background(), usersQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response, got %#v", resp)
	}
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabaseOrFallback(t *testing.T) {
	repo := NewPostgresRepository(nil)

	_, err := repo.ListGroups(context.Background(), groupsQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryFiltersUsersByRadiusAndExcludesSelf(t *testing.T) {
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedUsers: {
			{"u1", "self", "", "", "loc", 35.0, 112.0, false, false, 0, 0},
			{"u2", "near", "", "", "loc", 35.0001, 112.0, true, false, 2, 1},
			{"u3", "far", "", "", "loc", 36.0, 112.0, false, false, 0, 0},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListUsers(context.Background(), usersQuery{ViewerUserID: "u1", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].User.UserID != "u2" {
		t.Fatalf("expected only nearby non-self user, got %#v", resp)
	}
}

func TestPostgresRepositoryPaginatesUsersWithOpaqueCursor(t *testing.T) {
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedUsers: {
			{"u3", "c", "", "", "loc", 35.0001, 112.0, false, false, 0, 0},
			{"u2", "b", "", "", "loc", 35.0002, 112.0, false, false, 0, 0},
			{"u1", "a", "", "", "loc", 35.0003, 112.0, false, false, 0, 0},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))
	query := usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 2}

	first, err := repo.ListUsers(context.Background(), query)
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].User.UserID != "u3" || first.Items[1].User.UserID != "u2" || first.NextCursor == "" {
		t.Fatalf("expected next cursor, got %#v", first)
	}
	second, err := repo.ListUsers(context.Background(), usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("ListUsers second page error: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].User.UserID != "u1" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestPostgresRepositoryFiltersGroupsByRadiusAndSorts(t *testing.T) {
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedGroups: {
			{"g0", "far", "", "", false, "loc", 35.03, 112.0, 10, false, 0},
			{"g1", "near-a", "", "", false, "loc", 35.0001, 112.0, 10, true, 3},
			{"g2", "near-b", "", "", false, "loc", 35.0001, 112.0, 10, true, 3},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListGroups(context.Background(), groupsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 10})
	if err != nil {
		t.Fatalf("ListGroups returned error: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].Group.GroupID != "g2" || resp.Items[1].Group.GroupID != "g1" {
		t.Fatalf("expected near groups sorted by group_id desc on tie, got %#v", resp)
	}
}

func TestPostgresRepositoryLoadsFollowingMomentsOnly(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedMomentsFollowing: {
			{"m3", "u3", "nick", "", "newest", base.Add(2 * time.Hour), 5, 1, 0, "loc", 35.0, 112.0, true, false},
			{"m2", "u2", "nick", "", "new", base.Add(time.Hour), 5, 1, 0, "loc", 35.0, 112.0, true, false},
			{"m1", "u1", "nick", "", "old", base, 1, 0, 0, "loc", 35.0, 112.0, false, false},
		},
		queryFeedMomentTags: {{"hot"}},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", Source: "following", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].MomentID != "m3" || resp.Items[1].MomentID != "m2" || resp.Items[0].Recommendation == nil || resp.Items[0].Recommendation.Type != "following" || resp.NextCursor == "" {
		t.Fatalf("unexpected following moments: %#v", resp)
	}
	second, err := repo.ListMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", Source: "following", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2, Cursor: resp.NextCursor})
	if err != nil {
		t.Fatalf("ListMoments second page error: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].MomentID != "m1" {
		t.Fatalf("unexpected following second page: %#v", second)
	}
}

func TestPostgresRepositoryLoadsNearbyMomentsOnlyWithinRadius(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedMomentsNearby: {
			{"m1", "u1", "nick", "", "near", base, 1, 0, 0, "loc", 35.0001, 112.0, false, false, false},
			{"m2", "u2", "nick", "", "far", base, 99, 0, 0, "loc", 36.0, 112.0, false, false, false},
		},
		queryFeedMomentTags: {},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", Source: "nearby", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].MomentID != "m1" {
		t.Fatalf("expected only nearby moment, got %#v", resp)
	}
}

func TestPostgresRepositoryAllMomentsUsesStableCursorPagination(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedMomentsAll: {
			{"m1", "u1", "nick", "", "one", base, 10, 0, 0, "loc", 35.0001, 112.0, false, false, false},
			{"m2", "u2", "nick", "", "two", base.Add(time.Minute), 9, 0, 0, "loc", 35.0002, 112.0, false, false, true},
			{"m3", "u3", "nick", "", "three", base.Add(2 * time.Minute), 8, 0, 0, "loc", 35.0003, 112.0, false, false, false},
		},
		queryFeedMomentTags: {},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))
	query := momentsQuery{ViewerUserID: "viewer", Source: "all", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2}

	first, err := repo.ListMoments(context.Background(), query)
	if err != nil {
		t.Fatalf("ListMoments returned error: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("expected first page with cursor, got %#v", first)
	}
	second, err := repo.ListMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", Source: "all", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("ListMoments second page error: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestPostgresRepositoryTrendingAggregatesTagsAndMoments(t *testing.T) {
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	backend := &feedBackendStub{queryRowsByQuery: map[string][][]any{
		queryFeedTrendingTags:       {{"hot", 3}, {"new", 1}},
		queryFeedTrendingHotMoments: {{"m1", "u1", "nick", "", "desc", base, 9, 1, 0, "loc", 35.0, 112.0, false, false}},
		queryFeedMomentTags:         {{"hot"}},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListTrending(context.Background(), "viewer")
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(resp.TrendingTags) != 2 || len(resp.HotMoments) != 1 || resp.TrendingTags[0].HotMomentPreview == nil || resp.TrendingTags[0].HotMomentPreview.MomentID != "m1" {
		t.Fatalf("unexpected trending bundle: %#v", resp)
	}
}

func TestPostgresRepositoryRejectsInvalidCursor(t *testing.T) {
	_, err := decodeFeedCursor("not-a-cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestPostgresRepositoryPropagatesQueryError(t *testing.T) {
	repo := NewPostgresRepository(newFeedFactory(&feedBackendStub{queryErrs: map[string]error{queryFeedUsers: errors.New("boom")}}))

	_, err := repo.ListUsers(context.Background(), usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected query error, got %v", err)
	}
}

func TestPostgresRepositoryDismissReturnsNotImplementedWithoutDatabase(t *testing.T) {
	repo := NewPostgresRepository(nil)

	if err := repo.Dismiss(context.Background(), dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "m1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryDismissReturnsNotFoundWhenTargetMissing(t *testing.T) {
	backend := &feedBackendStub{queryRowByQuery: map[string]feedRowValues{queryFeedDismissMomentExists: {err: errors.New("no rows")}}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	if err := repo.Dismiss(context.Background(), dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "m1"}); !errors.Is(err, ErrDismissTargetNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPostgresRepositoryDismissInsertsWhenTargetExists(t *testing.T) {
	backend := &feedBackendStub{queryRowByQuery: map[string]feedRowValues{queryFeedDismissMomentExists: {values: []any{"m1"}}}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	if err := repo.Dismiss(context.Background(), dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "m1"}); err != nil {
		t.Fatalf("Dismiss returned error: %v", err)
	}
	if len(backend.execCalls) != 1 || backend.execCalls[0].query != queryFeedDismissInsert {
		t.Fatalf("unexpected exec calls: %#v", backend.execCalls)
	}
}

func TestPostgresRepositoryDismissPropagatesExecError(t *testing.T) {
	backend := &feedBackendStub{
		queryRowByQuery: map[string]feedRowValues{queryFeedDismissMomentExists: {values: []any{"m1"}}},
		execErrs:        map[string]error{queryFeedDismissInsert: errors.New("boom")},
	}
	repo := NewPostgresRepository(newFeedFactory(backend))

	if err := repo.Dismiss(context.Background(), dismissCommand{ViewerUserID: "u1", TargetType: "moment", TargetID: "m1"}); err == nil || err.Error() != "boom" {
		t.Fatalf("expected exec error, got %v", err)
	}
}
