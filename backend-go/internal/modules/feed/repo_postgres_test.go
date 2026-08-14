package feed

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type feedBackendStub struct {
	queryRowsByKey map[string][][]any
	queryErrs      map[string]error
}

func (b *feedBackendStub) Ping(context.Context) error { return nil }
func (b *feedBackendStub) Close() error               { return nil }
func (b *feedBackendStub) Exec(_ context.Context, _ string, _ ...any) error {
	return nil
}
func (b *feedBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *feedBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	for key, err := range b.queryErrs {
		if strings.Contains(query, key) {
			return nil, err
		}
	}
	for key, rows := range b.queryRowsByKey {
		if strings.Contains(query, key) {
			return &testdb.RowsValues{Rows: rows}, nil
		}
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

	resp, err := repo.ListNearbyUsers(context.Background(), usersQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty response, got %#v", resp)
	}
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabaseOrFallback(t *testing.T) {
	repo := NewPostgresRepository(nil)

	_, err := repo.ListNearbyGroups(context.Background(), groupsQuery{Limit: 10})
	if !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryFiltersUsersByRadiusAndExcludesSelf(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		"FROM users u": {
			{"u1", "self", "", "", "loc", 35.0, 112.0, false, false, 0, 0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0},
			{"u2", "near", "", "", "loc", 35.0001, 112.0, true, false, 2, 1, "online", time.Time{}, []string{"tag1"}, 3, 1, 2, 1, 1},
			{"u3", "far", "", "", "loc", 36.0, 112.0, false, false, 0, 0, "offline", time.Time{}, []string{}, 0, 0, 0, 0, 0},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListNearbyUsers(context.Background(), usersQuery{ViewerUserID: "u1", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err != nil {
		t.Fatalf("ListNearbyUsers returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].User.UserID != "u2" {
		t.Fatalf("expected only nearby non-self user, got %#v", resp)
	}
}

func TestPostgresRepositoryPaginatesUsersWithOpaqueCursor(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		"FROM users u": {
			{"u3", "c", "", "", "loc", 35.0001, 112.0, false, false, 0, 0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0},
			{"u2", "b", "", "", "loc", 35.0002, 112.0, false, false, 0, 0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0},
			{"u1", "a", "", "", "loc", 35.0003, 112.0, false, false, 0, 0, "online", time.Time{}, []string{}, 0, 0, 0, 0, 0},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))
	query := usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 2}

	first, err := repo.ListNearbyUsers(context.Background(), query)
	if err != nil {
		t.Fatalf("ListNearbyUsers returned error: %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].User.UserID != "u3" || first.Items[1].User.UserID != "u2" || first.NextCursor == nil {
		t.Fatalf("expected next cursor, got %#v", first)
	}
	second, err := repo.ListNearbyUsers(context.Background(), usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 2, Cursor: *first.NextCursor})
	if err != nil {
		t.Fatalf("ListNearbyUsers second page error: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].User.UserID != "u1" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestPostgresRepositoryFiltersGroupsByRadiusAndSorts(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		"FROM groups g": {
			{"g0", "far", "", "", false, "loc", 35.03, 112.0, 10, false, 0, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 2, "owner", time.Time{}, time.Time{}, false, 0},
			{"g1", "near-a", "", "", false, "loc", 35.0001, 112.0, 10, true, 3, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 1, "member", time.Time{}, time.Time{}, false, 0},
			{"g2", "near-b", "", "", false, "loc", 35.0001, 112.0, 10, true, 3, "direct", false, []string{}, "", true, 500, time.Time{}, "owner1", 1, "member", time.Time{}, time.Time{}, false, 0},
		},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListNearbyGroups(context.Background(), groupsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 2000, Limit: 10})
	if err != nil {
		t.Fatalf("ListNearbyGroups returned error: %v", err)
	}
	// spec api_spec_discovery.md：距离并列时 tie-breaker 为 group_id ASC
	if len(resp.Items) != 2 || resp.Items[0].Group.GroupID != "g1" || resp.Items[1].Group.GroupID != "g2" {
		t.Fatalf("expected near groups sorted by group_id asc on tie, got %#v", resp)
	}
}

func TestPostgresRepositoryLoadsFollowingMomentsOnly(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		queryFeedMomentsFollowing: {
			{"m3", "u3", "nick", "", "newest", time.Date(2026, 3, 3, 2, 0, 0, 0, time.UTC), 5, 1, 0, "loc", 35.0, 112.0, true, false},
			{"m2", "u2", "nick", "", "new", time.Date(2026, 3, 3, 1, 0, 0, 0, time.UTC), 5, 1, 0, "loc", 35.0, 112.0, true, false},
			{"m1", "u1", "nick", "", "old", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 1, 0, 0, "loc", 35.0, 112.0, false, false},
		},
		queryFeedMomentTags: {{"hot"}},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListFollowingMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Limit: 2})
	if err != nil {
		t.Fatalf("ListFollowingMoments returned error: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].MomentID != "m3" || resp.Items[1].MomentID != "m2" || resp.Items[0].Recommendation == nil || *resp.Items[0].Recommendation != "following" || resp.NextCursor == nil {
		t.Fatalf("unexpected following moments: %#v", resp)
	}
	second, err := repo.ListFollowingMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Limit: 2, Cursor: *resp.NextCursor})
	if err != nil {
		t.Fatalf("ListFollowingMoments second page error: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].MomentID != "m1" {
		t.Fatalf("unexpected following second page: %#v", second)
	}
}

func TestPostgresRepositoryLoadsNearbyMomentsOnlyWithinRadius(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		queryFeedMomentsNearby: {
			{"m1", "u1", "nick", "", "near", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 1, 0, 0, "loc", 35.0001, 112.0, false, false, false},
			{"m2", "u2", "nick", "", "far", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 99, 0, 0, "loc", 36.0, 112.0, false, false, false},
		},
		queryFeedMomentTags: {},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListNearbyMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err != nil {
		t.Fatalf("ListNearbyMoments returned error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].MomentID != "m1" {
		t.Fatalf("expected only nearby moment, got %#v", resp)
	}
}

func TestPostgresRepositoryNearbyMomentsUsesStableCursorPagination(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		queryFeedMomentsNearby: {
			{"m1", "u1", "nick", "", "one", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 10, 0, 0, "loc", 35.0001, 112.0, false, false, false},
			{"m2", "u2", "nick", "", "two", time.Date(2026, 3, 3, 1, 0, 0, 0, time.UTC), 9, 0, 0, "loc", 35.0002, 112.0, false, false, true},
			{"m3", "u3", "nick", "", "three", time.Date(2026, 3, 3, 2, 0, 0, 0, time.UTC), 8, 0, 0, "loc", 35.0003, 112.0, false, false, false},
		},
		queryFeedMomentTags: {},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))
	query := momentsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2}

	first, err := repo.ListNearbyMoments(context.Background(), query)
	if err != nil {
		t.Fatalf("ListNearbyMoments returned error: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatalf("expected first page with cursor, got %#v", first)
	}
	second, err := repo.ListNearbyMoments(context.Background(), momentsQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 5000, Limit: 2, Cursor: *first.NextCursor})
	if err != nil {
		t.Fatalf("ListNearbyMoments second page error: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestPostgresRepositoryTrendingLoadsHotMoments(t *testing.T) {
	backend := &feedBackendStub{queryRowsByKey: map[string][][]any{
		queryFeedTrendingHotMoments:    {{"m1", "u1", "nick", "", "desc", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), 9, 1, 0, "loc", 35.0, 112.0, false, false}},
		queryFeedMomentTags:            {{"hot"}},
		queryFeedMomentTagsByMomentIDs: {{"m1", "hot"}},
	}}
	repo := NewPostgresRepository(newFeedFactory(backend))

	resp, err := repo.ListTrending(context.Background(), "viewer")
	if err != nil {
		t.Fatalf("ListTrending returned error: %v", err)
	}
	if len(resp.HotMoments) != 1 {
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
	repo := NewPostgresRepository(newFeedFactory(&feedBackendStub{queryErrs: map[string]error{"FROM users u": errors.New("boom")}}))

	_, err := repo.ListNearbyUsers(context.Background(), usersQuery{ViewerUserID: "viewer", ReferencePoint: GeoPoint{Latitude: 35, Longitude: 112}, Radius: 1000, Limit: 10})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected query error, got %v", err)
	}
}
