package users

import (
	"context"
	"errors"
	"testing"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type userRowValues struct {
	values []any
	err    error
}

func (r userRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *float64:
			*d = r.values[i].(float64)
		case *bool:
			*d = r.values[i].(bool)
		}
	}
	return nil
}

type userRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *userRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *userRowsValues) Scan(dest ...any) error {
	row := userRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *userRowsValues) Close() error { return nil }
func (r *userRowsValues) Err() error   { return r.err }

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
func (r *userFallbackRepo) SearchProfiles(context.Context, SearchUsersRequest) ([]MyProfile, error) {
	r.markUsed()
	return nil, nil
}
func (r *userFallbackRepo) Follow(context.Context, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) Unfollow(context.Context, string, string) error {
	r.markUsed()
	return nil
}
func (r *userFallbackRepo) Block(context.Context, string, string) error {
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
func (r *userFallbackRepo) ListBlocked(context.Context, string) ([]BlockedUserItem, error) {
	r.markUsed()
	return nil, nil
}
func (r *userFallbackRepo) ListLikedMoments(context.Context, string, int, string) (momenttypes.PaginatedMoments, error) {
	r.markUsed()
	return momenttypes.PaginatedMoments{}, nil
}
func (r *userFallbackRepo) Relationship(context.Context, string, string) (Relationship, error) {
	r.markUsed()
	return RelationshipNone, nil
}

type userBackendStub struct {
	queryRowByQuery map[string]persistence.Row
	queryRowsByQuery map[string][][]any
	queryErrs map[string]error
	execErrs map[string]error
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
	return userRowValues{err: persistence.ErrNotImplemented}
}
func (b *userBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if rows, ok := b.queryRowsByQuery[query]; ok {
		return &userRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryReadsFromQueryableDB(t *testing.T) {
	backend := &userBackendStub{
		queryRowByQuery: map[string]persistence.Row{
			queryGetUserProfile:  userRowValues{values: []any{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", "2026-03-20T00:00:00Z", "2026-03-19T00:00:00Z"}},
			queryGetUserSettings: userRowValues{values: []any{"public", true, true, true, true, true, true, true, "everyone", true, false, true, true}},
			queryGetUserRelation: userRowValues{values: []any{true, false, "buddy"}},
		},
		queryRowsByQuery: map[string][][]any{
			querySearchUserProfiles: {{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", "2026-03-20T00:00:00Z", "2026-03-19T00:00:00Z"}},
			queryListBlockedUsers:  {{"user_bob", "buddy"}},
			queryListLikedMoments:  {{"moment_1", "user_alice", "hello", "2026-03-20T00:00:00Z", 3, 2, 1, "南天门", 35.0, 112.0, "public", "2026-03-21T00:00:00Z"}},
			momenttypesQueryListMomentTagsByMomentID: {{"travel"}, {"hotspot"}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &userFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	profile, ok, err := repo.GetProfile(context.Background(), "user_alice")
	if err != nil || !ok || profile.UserID != "user_alice" || profile.Nickname != "Alice" {
		t.Fatalf("unexpected profile: %#v ok=%v err=%v", profile, ok, err)
	}
	settings, ok, err := repo.GetSettings(context.Background(), "user_alice")
	if err != nil || !ok || settings.AllowPrivateChat != PrivateChatEveryone {
		t.Fatalf("unexpected settings: %#v ok=%v err=%v", settings, ok, err)
	}
	profiles, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{Keyword: "Alice"})
	if err != nil || len(profiles) != 1 || profiles[0].UserID != "user_alice" {
		t.Fatalf("unexpected search result: %#v err=%v", profiles, err)
	}
	blocked, err := repo.ListBlocked(context.Background(), "user_alice")
	if err != nil || len(blocked) != 1 || blocked[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected blocked users: %#v err=%v", blocked, err)
	}
	rel, err := repo.Relationship(context.Background(), "user_alice", "user_bob")
	if err != nil || rel != RelationshipFollowing {
		t.Fatalf("unexpected relationship: %v err=%v", rel, err)
	}
	liked, err := repo.ListLikedMoments(context.Background(), "user_alice", 20, "")
	if err != nil || len(liked.Items) != 1 || liked.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected liked moments: %#v err=%v", liked, err)
	}
	if liked.Items[0].MyState == nil || !liked.Items[0].MyState.IsLiked || len(liked.Items[0].Tags) != 2 {
		t.Fatalf("unexpected liked moment state: %#v", liked.Items[0])
	}
	if fallback.used {
		t.Fatal("expected postgres path without fallback")
	}
}

func TestPostgresRepositorySearchAndListBlockedReturnErrorsWithoutFallback(t *testing.T) {
	backend := &userBackendStub{queryRowByQuery: map[string]persistence.Row{}, queryRowsByQuery: map[string][][]any{}, queryErrs: map[string]error{querySearchUserProfiles: errors.New("boom"), queryListBlockedUsers: errors.New("boom")}, execErrs: map[string]error{}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &userFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	if _, err := repo.SearchProfiles(context.Background(), SearchUsersRequest{Keyword: "Alice"}); err == nil {
		t.Fatal("expected search error")
	}
	if _, err := repo.ListBlocked(context.Background(), "user_alice"); err == nil {
		t.Fatal("expected list blocked error")
	}
	backend.queryErrs[queryListLikedMoments] = errors.New("boom")
	if _, err := repo.ListLikedMoments(context.Background(), "user_alice", 20, ""); err == nil {
		t.Fatal("expected liked moments error")
	}
	if fallback.used {
		t.Fatal("expected query failures without fallback")
	}
}

func TestPostgresRepositoryWritesReturnExecErrors(t *testing.T) {
	backend := &userBackendStub{queryRowByQuery: map[string]persistence.Row{queryGetUserRelation: userRowValues{values: []any{false, false, ""}}}, queryRowsByQuery: map[string][][]any{}, queryErrs: map[string]error{}, execErrs: map[string]error{queryUpsertUserProfile: errors.New("boom")}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &userFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	if err := repo.PutProfile(context.Background(), MyProfile{UserID: "user_alice", Nickname: "Alice", UserType: UserTypeNormal, Location: Location{LocationName: "南天门"}}); err == nil {
		t.Fatal("expected put profile error")
	}
	backend.execErrs = map[string]error{queryUpsertUserRelation: errors.New("boom")}
	if err := repo.Follow(context.Background(), "user_alice", "user_bob"); err == nil {
		t.Fatal("expected follow error")
	}
	if fallback.used {
		t.Fatal("expected write failures without fallback")
	}
}
