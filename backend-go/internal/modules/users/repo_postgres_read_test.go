package users

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"strings"
	"testing"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type userFallbackRepo struct{ used bool }

func (r *userFallbackRepo) markUsed() { r.used = true }
func (r *userFallbackRepo) GetProfile(context.Context, string) (MyProfile, bool, error) {
	r.markUsed()
	return MyProfile{}, false, nil
}
func (r *userFallbackRepo) PutProfile(context.Context, MyProfile) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) DeleteProfile(context.Context, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) GetSettings(context.Context, string) (UserSettings, bool, error) {
	r.markUsed()
	return UserSettings{}, false, nil
}
func (r *userFallbackRepo) PutSettings(context.Context, string, UserSettings) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error) {
	r.markUsed()
	return PaginatedProfiles{}, nil
}
func (r *userFallbackRepo) Follow(context.Context, string, string, FollowSource, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) Unfollow(context.Context, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) Block(context.Context, string, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) Unblock(context.Context, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) SetRemark(context.Context, string, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) ListBlocked(context.Context, string, int, string) (PaginatedBlockedUsers, error) {
	r.markUsed()
	return PaginatedBlockedUsers{}, nil
}
func (r *userFallbackRepo) ListReactedMoments(context.Context, string, int, string, string) (momenttypes.PaginatedMoments, error) {
	r.markUsed()
	return momenttypes.PaginatedMoments{}, nil
}
func (r *userFallbackRepo) Relationship(context.Context, string, string) (Relationship, error) {
	r.markUsed()
	return Relationship{}, nil
}
func (r *userFallbackRepo) RelationshipBatch(context.Context, string, []string) (map[string]Relationship, error) {
	r.markUsed()
	return map[string]Relationship{}, nil
}
func (r *userFallbackRepo) ListFriends(context.Context, string, int, string) (PaginatedFriends, error) {
	r.markUsed()
	return PaginatedFriends{}, nil
}
func (r *userFallbackRepo) ListFollowing(context.Context, string, int, string) (PaginatedFollowing, error) {
	r.markUsed()
	return PaginatedFollowing{}, nil
}
func (r *userFallbackRepo) ListFollowers(context.Context, string, int, string) (PaginatedFollowers, error) {
	r.markUsed()
	return PaginatedFollowers{}, nil
}
func (r *userFallbackRepo) ListGroups(context.Context, string, int, string) (grouptypes.PaginatedGroups, error) {
	r.markUsed()
	return grouptypes.PaginatedGroups{}, nil
}
func (r *userFallbackRepo) ListMoments(context.Context, string, int, string, bool) (momenttypes.PaginatedMoments, error) {
	r.markUsed()
	return momenttypes.PaginatedMoments{}, nil
}
func (r *userFallbackRepo) CommonGroupCount(context.Context, string, string) (int, error) {
	r.markUsed()
	return 0, nil
}
func (r *userFallbackRepo) UserExists(context.Context, string) (bool, error) {
	r.markUsed()
	return false, nil
}

type userBackendStub struct {
	queryRowByQuery  map[string]persistence.Row
	queryRowsByQuery map[string][][]any
	queryErrs        map[string]error
	execErrs         map[string]error
}

func (b *userBackendStub) Ping(context.Context) error { return nil }
func (b *userBackendStub) Close() error               { return nil }
func (b *userBackendStub) Exec(_ context.Context, query string, _ ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	return nil
}
func (b *userBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if row, ok := b.queryRowByQuery[query]; ok {
		return row
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *userBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err, ok := b.queryErrs[query]; ok {
		return nil, err
	}
	for key, err := range b.queryErrs {
		if key != "" && strings.Contains(query, key) {
			return nil, err
		}
	}
	if rows, ok := b.queryRowsByQuery[query]; ok {
		return &testdb.RowsValues{Rows: rows}, nil
	}
	for key, rows := range b.queryRowsByQuery {
		if key != "" && strings.Contains(query, key) {
			return &testdb.RowsValues{Rows: rows}, nil
		}
	}
	return &persistence.InMemoryRows{}, nil
}

type userMomentCursorBackend struct {
	firstPage  [][]any
	secondPage [][]any
	tags       [][]any
}

func (b *userMomentCursorBackend) Ping(context.Context) error                 { return nil }
func (b *userMomentCursorBackend) Close() error                               { return nil }
func (b *userMomentCursorBackend) Exec(context.Context, string, ...any) error { return nil }
func (b *userMomentCursorBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *userMomentCursorBackend) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	switch query {
	case queryListUserMoments:
		if len(args) >= 4 && args[2] != nil && args[3] == "moment_2" {
			return &testdb.RowsValues{Rows: b.secondPage}, nil
		}
		return &testdb.RowsValues{Rows: b.firstPage}, nil
	case momenttypesQueryListMomentTagsByMomentID, momenttypesQueryListMomentTagsByMomentIDs:
		return &testdb.RowsValues{Rows: b.tags}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestPostgresRepositoryReadsFromQueryableDB(t *testing.T) {
	backend := &userBackendStub{
		queryRowByQuery: map[string]persistence.Row{
			queryGetUserProfile:  testdb.RowValues{Values: []any{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), false}},
			queryGetUserSettings: testdb.RowValues{Values: []any{"public", true, true, true, true, true, true, true, "everyone", true, false, true, true, false}},
			queryGetUserRelation: testdb.RowValues{Values: []any{true, false, "buddy", time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC), nil, ""}},
			queryCountUserStats:  testdb.RowValues{Values: []any{2, 1, 3, 4, 5}},
		},
		queryRowsByQuery: map[string][][]any{
			"FROM users u":                            {{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), false}},
			queryListBlockedUsers:                     {{"user_bob", time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC), "buddy"}},
			queryListReactedMoments:                   {{"moment_1", "user_alice", "Alice", "hello", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), 3, 2, 1, "南天门", 35.0, 112.0, "public", time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)}},
			momenttypesQueryListMomentTagsByMomentID:  {{"travel"}, {"hotspot"}},
			momenttypesQueryListMomentTagsByMomentIDs: {{"moment_1", "travel"}, {"moment_1", "hotspot"}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	profile, ok, err := repo.GetProfile(context.Background(), "user_alice")
	if err != nil || !ok || profile.UserID != "user_alice" || profile.Nickname != "Alice" {
		t.Fatalf("unexpected profile: %#v ok=%v err=%v", profile, ok, err)
	}
	settings, ok, err := repo.GetSettings(context.Background(), "user_alice")
	if err != nil || !ok || settings.AllowPrivateChat != ChatPermEveryone {
		t.Fatalf("unexpected settings: %#v ok=%v err=%v", settings, ok, err)
	}
	profiles, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{Keyword: "Alice", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}})
	if err != nil || len(profiles.Items) != 1 || profiles.Items[0].UserID != "user_alice" {
		t.Fatalf("unexpected search result: %#v err=%v", profiles, err)
	}
	blocked, err := repo.ListBlocked(context.Background(), "user_alice", 20, "")
	if err != nil || len(blocked.Items) != 1 || blocked.Items[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected blocked users: %#v err=%v", blocked, err)
	}
	rel, err := repo.Relationship(context.Background(), "user_alice", "user_bob")
	if err != nil || !rel.IsFollowing || rel.IsBlocked {
		t.Fatalf("unexpected relationship: %v err=%v", rel, err)
	}
	liked, err := repo.ListReactedMoments(context.Background(), "user_alice", 20, "", "")
	if err != nil || len(liked.Items) != 1 || liked.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected reacted moments: %#v err=%v", liked, err)
	}
	if liked.Items[0].MyState == nil || len(liked.Items[0].Tags) != 2 {
		t.Fatalf("unexpected liked moment state: %#v", liked.Items[0])
	}
}

func TestPostgresRepositorySearchAndListBlockedReturnErrorsWithoutFallback(t *testing.T) {
	backend := &userBackendStub{queryRowByQuery: map[string]persistence.Row{}, queryRowsByQuery: map[string][][]any{}, queryErrs: map[string]error{"FROM users u": errors.New("boom"), queryListBlockedUsers: errors.New("boom")}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	if _, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{Keyword: "Alice", ReferencePoint: GeoPoint{Latitude: 35.0, Longitude: 112.0}}); err == nil {
		t.Fatal("expected search error")
	}
	if _, err := repo.ListBlocked(context.Background(), "user_alice", 20, ""); err == nil {
		t.Fatal("expected list blocked error")
	}
	backend.queryErrs[queryListReactedMoments] = errors.New("boom")
	if _, err := repo.ListReactedMoments(context.Background(), "user_alice", 20, "", ""); err == nil {
		t.Fatal("expected reacted moments error")
	}
}

func TestPostgresRepositoryWritesReturnExecErrors(t *testing.T) {
	backend := &userBackendStub{queryRowByQuery: map[string]persistence.Row{queryGetUserRelation: testdb.RowValues{Values: []any{false, false, "", nil, nil, ""}}}, queryRowsByQuery: map[string][][]any{}, queryErrs: map[string]error{}, execErrs: map[string]error{queryUpsertUserProfile: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	if err := repo.PutProfile(context.Background(), MyProfile{UserID: "user_alice", Nickname: "Alice", UserType: UserTypeNormal, Location: Location{LocationName: "南天门"}}); err == nil {
		t.Fatal("expected put profile error")
	}
	backend.execErrs = map[string]error{queryUpsertUserRelationFollow: errors.New("boom")}
	if err := repo.Follow(context.Background(), "user_alice", "user_bob", FollowSourceProfile, ""); err == nil {
		t.Fatal("expected follow error")
	}
}

func TestUserMomentCursorIsOpaqueAndTamperProof(t *testing.T) {
	cursor := encodeUserMomentCursor("2026-03-29T12:00:00Z", "moment_1")
	if cursor == "" {
		t.Fatal("expected cursor")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		t.Fatalf("decode cursor token: %v", err)
	}
	var plain map[string]any
	if err := json.Unmarshal(decoded, &plain); err == nil {
		t.Fatalf("expected opaque cursor, got plain json payload %#v", plain)
	}
	parsedAt, parsedID, err := decodeUserMomentCursor(cursor)
	if err != nil {
		t.Fatalf("decode user moment cursor: %v", err)
	}
	if parsedID != "moment_1" {
		t.Fatalf("unexpected parsed id: %s", parsedID)
	}
	at, ok := parsedAt.(time.Time)
	if !ok || at.UTC().Format(time.RFC3339) != "2026-03-29T12:00:00Z" {
		t.Fatalf("unexpected parsed at: %#v", parsedAt)
	}
	tampered := cursor[:len(cursor)-2] + "AA"
	if tampered == cursor {
		t.Fatal("expected tampered cursor to differ from original")
	}
	if _, _, err := decodeUserMomentCursor(tampered); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestListUserMomentsPostgresReturnsNextCursorWithoutDupingNextPage(t *testing.T) {
	createdAt1 := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2026, 3, 29, 11, 0, 0, 0, time.UTC)
	createdAt3 := time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)
	backend := &userMomentCursorBackend{
		firstPage: [][]any{
			{"moment_3", "user_alice", "Alice", "third", createdAt3, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
			{"moment_2", "user_alice", "Alice", "second", createdAt2, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
			{"moment_1", "user_alice", "Alice", "first", createdAt1, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
		},
		secondPage: [][]any{
			{"moment_1", "user_alice", "Alice", "first", createdAt1, 0, 0, 0, "南天门", 35.0, 112.0, "public"},
		},
		tags: [][]any{{"moment_3", "travel"}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend

	page1, err := listUserMomentsPostgres(context.Background(), factory, "user_alice", 2, "", true)
	if err != nil {
		t.Fatalf("list first user moments page: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("expected 2 items, got %#v", page1.Items)
	}
	if page1.Items[0].MomentID != "moment_3" || page1.Items[1].MomentID != "moment_2" {
		t.Fatalf("unexpected first page ids: %#v", page1.Items)
	}
	if page1.NextCursor == "" {
		t.Fatal("expected next cursor")
	}
	parsedAt, parsedID, err := decodeUserMomentCursor(page1.NextCursor)
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}
	cursorAt, ok := parsedAt.(time.Time)
	if !ok {
		t.Fatalf("expected cursor time, got %#v", parsedAt)
	}
	if parsedID != "moment_2" || cursorAt.UTC().Format(time.RFC3339) != createdAt2.Format(time.RFC3339) {
		t.Fatalf("unexpected next cursor anchor: %v %s", cursorAt, parsedID)
	}

	page2, err := listUserMomentsPostgres(context.Background(), factory, "user_alice", 2, page1.NextCursor, true)
	if err != nil {
		t.Fatalf("list second user moments page: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected second page items: %#v", page2.Items)
	}
	if page2.NextCursor != "" {
		t.Fatalf("expected empty next cursor on second page, got %q", page2.NextCursor)
	}
	for _, item := range page2.Items {
		if item.MomentID == "moment_2" || item.MomentID == "moment_3" {
			t.Fatalf("expected no duplicate items across pages, got %#v", page2.Items)
		}
	}
}

func TestListUserMomentsPostgresRejectsInvalidCursor(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &userBackendStub{queryRowByQuery: map[string]persistence.Row{}, queryRowsByQuery: map[string][][]any{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}

	_, err := listUserMomentsPostgres(context.Background(), factory, "user_alice", 20, "bad-cursor", true)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}
