package users

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"

	"github.com/jackc/pgx/v5"
)

// g3Backend 是 persistence.Queryable 的通用可注入桩：Query 支持精确键与
// 子串键匹配（动态 SQL 用子串定位），QueryRow 精确匹配。
type g3Backend struct {
	queryRowByQuery map[string]persistence.Row
	rowsByQuery     map[string]persistence.Rows
	queryErrs       map[string]error
	execErrs        map[string]error
	execCount       int
}

func (b *g3Backend) Ping(context.Context) error { return nil }
func (b *g3Backend) Close() error               { return nil }

func (b *g3Backend) Exec(_ context.Context, query string, _ ...any) error {
	b.execCount++
	if err, ok := b.execErrs[query]; ok {
		return err
	}
	return nil
}

func (b *g3Backend) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if row, ok := b.queryRowByQuery[query]; ok {
		return row
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}

func (b *g3Backend) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err, ok := b.queryErrs[query]; ok {
		return nil, err
	}
	for key, err := range b.queryErrs {
		if key != "" && strings.Contains(query, key) {
			return nil, err
		}
	}
	if rows, ok := b.rowsByQuery[query]; ok {
		return rows, nil
	}
	for key, rows := range b.rowsByQuery {
		if key != "" && strings.Contains(query, key) {
			return rows, nil
		}
	}
	return &persistence.InMemoryRows{}, nil
}

// g3Rows 是 persistence.Rows 的可注入桩：可模拟 Scan 错误、rows.Err 与普通行。
type g3Rows struct {
	rows    [][]any
	idx     int
	scanErr error
	tailErr error
}

func (r *g3Rows) Next() bool { return r.idx < len(r.rows) }
func (r *g3Rows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := testdb.RowValues{Values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *g3Rows) Close() error { return nil }
func (r *g3Rows) Err() error   { return r.tailErr }

func g3RepoWith(backend persistence.Queryable) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://g3"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return NewPostgresRepository(factory)
}

func g3EmptyRepo() *PostgresRepository {
	return NewPostgresRepository(persistence.NewFactory(config.Config{}))
}

var (
	g3Time     = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	g3TimeB    = time.Date(2026, 1, 1, 3, 4, 5, 0, time.UTC)
	g3StatsRow = []any{1, 2, 3, 4, 5}
)

func g3ProfileDetailsRow(userID string, tags []string) []any {
	row := []any{userID, "Nick", "", "normal", "bio", "male", "", "", "loc", 34.2, 117.1, "online", g3Time, g3Time, false}
	row = append(row, g3StatsRow...)
	row = append(row, tags)
	return row
}

func g3ProfileRow(userID string) []any {
	return []any{userID, "Nick", "", "normal", "bio", "male", "", "", "loc", 34.2, 117.1, "online", g3Time, g3Time, false}
}

func g3SettingsRow(allowPrivateChat string) []any {
	return []any{"public", true, true, true, true, true, true, true, allowPrivateChat, true, false, true, true, false}
}

// g3SearchRow 构造 scanSearchProfileRow 需要的行：15 画像列 (+可选 tags) + 3 可见性列。
func g3SearchRow(userID string, lat, lng float64, tags []string, includeTags bool, presence, location, stats bool) []any {
	row := []any{userID, "Nick", "", "normal", "bio", "male", "", "", "loc", lat, lng, "online", g3Time, g3Time, false}
	if includeTags {
		row = append(row, tags)
	}
	return append(row, presence, location, stats)
}

func TestG3RepoProfileReadBranches(t *testing.T) {
	noRows := errors.New("no rows")
	_ = noRows

	// GetProfile：ErrNoRows → found=false；其他错误 → 上抛；成功行（tags=nil 归一为空列表）。
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{
			queryGetUserProfileDetails: testdb.RowValues{Values: g3ProfileDetailsRow("u1", []string(nil))},
			queryGetRecentLocation:     testdb.RowValues{Values: []any{34.2, 117.1}},
			queryGetUserSettings:       testdb.RowValues{Values: g3SettingsRow("everyone")},
		},
		rowsByQuery: map[string]persistence.Rows{},
		queryErrs:   map[string]error{},
		execErrs:    map[string]error{},
	}
	repo := g3RepoWith(backend)
	profile, found, err := repo.GetProfile(context.Background(), "u1")
	if err != nil || !found || profile.UserID != "u1" || profile.Tags == nil || len(profile.Tags) != 0 {
		t.Fatalf("GetProfile success = %#v found=%v err=%v", profile, found, err)
	}
	if profile.Stats.MomentsCount != 1 || profile.Stats.GroupsCount != 5 {
		t.Fatalf("profile stats = %#v", profile.Stats)
	}
	point, found, err := repo.GetRecentLocation(context.Background(), "u1")
	if err != nil || !found || point.Latitude != 34.2 {
		t.Fatalf("GetRecentLocation success = %#v found=%v err=%v", point, found, err)
	}
	settings, found, err := repo.GetSettings(context.Background(), "u1")
	if err != nil || !found || settings.AllowPrivateChat != ChatPermEveryone {
		t.Fatalf("GetSettings success = %#v found=%v err=%v", settings, found, err)
	}

	backend.queryRowByQuery = map[string]persistence.Row{
		queryGetUserProfileDetails: testdb.RowValues{Err: pgx.ErrNoRows},
		queryGetRecentLocation:     testdb.RowValues{Err: pgx.ErrNoRows},
		queryGetUserSettings:       testdb.RowValues{Err: pgx.ErrNoRows},
	}
	if _, found, err := repo.GetProfile(context.Background(), "missing"); err != nil || found {
		t.Fatalf("GetProfile no rows found=%v err=%v", found, err)
	}
	if _, found, err := repo.GetRecentLocation(context.Background(), "missing"); err != nil || found {
		t.Fatalf("GetRecentLocation no rows found=%v err=%v", found, err)
	}
	if _, found, err := repo.GetSettings(context.Background(), "missing"); err != nil || found {
		t.Fatalf("GetSettings no rows found=%v err=%v", found, err)
	}

	backend.queryRowByQuery = map[string]persistence.Row{
		queryGetUserProfileDetails: testdb.RowValues{Err: errors.New("db down")},
		queryGetRecentLocation:     testdb.RowValues{Err: errors.New("db down")},
		queryGetUserSettings:       testdb.RowValues{Err: errors.New("db down")},
	}
	if _, _, err := repo.GetProfile(context.Background(), "u1"); err == nil {
		t.Fatal("expected GetProfile error")
	}
	if _, _, err := repo.GetRecentLocation(context.Background(), "u1"); err == nil {
		t.Fatal("expected GetRecentLocation error")
	}
	if _, _, err := repo.GetSettings(context.Background(), "u1"); err == nil {
		t.Fatal("expected GetSettings error")
	}

	if _, err := g3EmptyRepo().GetProfiles(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected GetProfiles not-implemented error")
	}
	if _, err := g3EmptyRepo().GetSettingsBatch(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected GetSettingsBatch not-implemented error")
	}
}

func TestG3RepoGetProfilesAndTagsBranches(t *testing.T) {
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryGetUserProfiles: &g3Rows{rows: [][]any{g3ProfileRow("u1"), g3ProfileRow("u2")}},
			queryListUserProfileTagsBatch: &g3Rows{rows: [][]any{
				{"u1", "tag_a"}, {"u1", "tag_b"}, {"u2", "tag_c"},
			}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	repo := g3RepoWith(backend)
	profiles, err := repo.GetProfiles(context.Background(), []string{"u1", "u2"})
	if err != nil {
		t.Fatalf("GetProfiles: %v", err)
	}
	if len(profiles) != 2 || len(profiles["u1"].Tags) != 2 || profiles["u2"].Tags[0] != "tag_c" {
		t.Fatalf("profiles = %#v", profiles)
	}
	if profiles["u1"].PresenceStatus != UserPresenceOnline || profiles["u1"].Location.Point.Latitude != 34.2 {
		t.Fatalf("scanned profile = %#v", profiles["u1"])
	}
	// 空 ID 列表直接返回空 map（不查库）。
	if result, err := repo.GetProfiles(context.Background(), nil); err != nil || len(result) != 0 {
		t.Fatalf("GetProfiles empty = %#v err=%v", result, err)
	}

	backend.queryErrs = map[string]error{queryGetUserProfiles: errors.New("boom")}
	if _, err := repo.GetProfiles(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected GetProfiles query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryGetUserProfiles] = &g3Rows{rows: [][]any{g3ProfileRow("u1")}, scanErr: errors.New("scan boom")}
	if _, err := repo.GetProfiles(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected GetProfiles scan error")
	}
	backend.rowsByQuery[queryGetUserProfiles] = &g3Rows{rows: [][]any{g3ProfileRow("u1")}, tailErr: errors.New("tail boom")}
	if _, err := repo.GetProfiles(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected GetProfiles rows error")
	}
	backend.rowsByQuery[queryGetUserProfiles] = &g3Rows{rows: [][]any{g3ProfileRow("u1")}}
	backend.queryErrs = map[string]error{queryListUserProfileTagsBatch: errors.New("tags boom")}
	if _, err := repo.GetProfiles(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected tags batch error")
	}

	// loadUserProfileTagsBatch 自身分支。
	tagsBackend := &g3Backend{rowsByQuery: map[string]persistence.Rows{
		queryListUserProfileTagsBatch: &g3Rows{rows: [][]any{{"u1", "x"}}},
	}, queryRowByQuery: map[string]persistence.Row{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}
	tags, err := loadUserProfileTagsBatch(context.Background(), tagsBackend, []string{"u1"})
	if err != nil || len(tags["u1"]) != 1 {
		t.Fatalf("tags batch = %#v err=%v", tags, err)
	}
	if result, err := loadUserProfileTagsBatch(context.Background(), tagsBackend, nil); err != nil || len(result) != 0 {
		t.Fatalf("tags batch empty = %#v err=%v", result, err)
	}
	tagsBackend.queryErrs = map[string]error{queryListUserProfileTagsBatch: errors.New("boom")}
	if _, err := loadUserProfileTagsBatch(context.Background(), tagsBackend, []string{"u1"}); err == nil {
		t.Fatal("expected tags query error")
	}
	tagsBackend.queryErrs = map[string]error{}
	tagsBackend.rowsByQuery[queryListUserProfileTagsBatch] = &g3Rows{rows: [][]any{{"u1", "x"}}, scanErr: errors.New("scan")}
	if _, err := loadUserProfileTagsBatch(context.Background(), tagsBackend, []string{"u1"}); err == nil {
		t.Fatal("expected tags scan error")
	}
	tagsBackend.rowsByQuery[queryListUserProfileTagsBatch] = &g3Rows{rows: [][]any{{"u1", "x"}}, tailErr: errors.New("tail")}
	if _, err := loadUserProfileTagsBatch(context.Background(), tagsBackend, []string{"u1"}); err == nil {
		t.Fatal("expected tags rows error")
	}
}

func TestG3RepoGetSettingsBatchBranches(t *testing.T) {
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryGetUserSettingsBatch: &g3Rows{rows: [][]any{
				append([]any{"u1"}, g3SettingsRow("friends")...),
				append([]any{"u2"}, g3SettingsRow("nobody")...),
			}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	repo := g3RepoWith(backend)
	settings, err := repo.GetSettingsBatch(context.Background(), []string{"u1", "u2"})
	if err != nil || len(settings) != 2 || settings["u1"].AllowPrivateChat != ChatPermFriends {
		t.Fatalf("settings batch = %#v err=%v", settings, err)
	}
	if result, err := repo.GetSettingsBatch(context.Background(), nil); err != nil || len(result) != 0 {
		t.Fatalf("settings batch empty = %#v err=%v", result, err)
	}
	backend.queryErrs = map[string]error{queryGetUserSettingsBatch: errors.New("boom")}
	if _, err := repo.GetSettingsBatch(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected settings batch query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryGetUserSettingsBatch] = &g3Rows{rows: [][]any{append([]any{"u1"}, g3SettingsRow("everyone")...)}, scanErr: errors.New("scan")}
	if _, err := repo.GetSettingsBatch(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected settings batch scan error")
	}
	backend.rowsByQuery[queryGetUserSettingsBatch] = &g3Rows{rows: [][]any{append([]any{"u1"}, g3SettingsRow("everyone")...)}, tailErr: errors.New("tail")}
	if _, err := repo.GetSettingsBatch(context.Background(), []string{"u1"}); err == nil {
		t.Fatal("expected settings batch rows error")
	}
}
