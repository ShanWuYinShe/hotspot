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
func (r *userFallbackRepo) BlockCascade(context.Context, string, string, string) error {
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
func (r *userFallbackRepo) CheckBlockedTags(context.Context, []string) ([]string, error) {
	r.markUsed()
	return nil, nil
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

func TestPostgresRepositoryReadsFromQueryableDB(t *testing.T) {
	backend := &userBackendStub{
		queryRowByQuery: map[string]persistence.Row{
			queryGetUserProfile:  testdb.RowValues{Values: []any{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), false}},
			queryGetUserSettings: testdb.RowValues{Values: []any{"public", true, true, true, true, true, true, true, "everyone", true, false, true, true, false}},
			queryGetUserRelation: testdb.RowValues{Values: []any{true, false, "buddy", time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC), nil, ""}},
			queryCountUserStats:  testdb.RowValues{Values: []any{2, 1, 3, 4, 5}},
		},
		queryRowsByQuery: map[string][][]any{
			"FROM users u":        {{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), false}},
			queryListBlockedUsers: {{"user_bob", time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC), "buddy"}},
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
