package users

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

// userListStub 按调用顺序弹出多组 rows（List*/RelationshipBatch 需要），
// 并支持 Query 级错误与固定 QueryRow 结果。
type userListStub struct {
	rowsSeq  []persistence.Rows
	row      persistence.Row
	queryErr error
}

func (s *userListStub) Exec(context.Context, string, ...any) error { return nil }
func (s *userListStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return s.row
}
func (s *userListStub) Query(context.Context, string, ...any) (persistence.Rows, error) {
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	if len(s.rowsSeq) == 0 {
		return &persistence.InMemoryRows{}, nil
	}
	rows := s.rowsSeq[0]
	s.rowsSeq = s.rowsSeq[1:]
	return rows, nil
}

// listRowsValues 与 userRowsValues 相同，另支持单行 Scan 错误与行尾 Err。
type listRowsValues struct {
	rows    [][]any
	idx     int
	err     error
	scanErr error
}

func (r *listRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *listRowsValues) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := testdb.RowValues{Values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *listRowsValues) Close() error { return nil }
func (r *listRowsValues) Err() error   { return r.err }

func friendRow(uid string, at time.Time) []any {
	return []any{uid, "昵称" + uid, "", "online", at, at, "", at.Add(-time.Hour), at.Add(-time.Minute)}
}

// followingRow/followerRow：is_mutual 列已随 RelatedUserItem 契约变更删除。
func followingRow(uid string, at time.Time) []any {
	return []any{uid, "昵称" + uid, "", at, "offline", at, at}
}

func followerRow(uid string, at time.Time) []any {
	return []any{uid, "昵称" + uid, "", at, "offline", at, at}
}

func groupRow(gid string, at time.Time) []any {
	return []any{gid, "群" + gid, "", "描述", true, "位置", 1.5, 2.5, "invite_only", true, 200, at}
}

func newUserListRepo(stub *userListStub) *PostgresRepository {
	return NewPostgresRepository(newUserFactory(stub))
}

func TestListFriendsPostgresPaginationAndErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Round(time.Second)

	// 3 行 + limit=2 → 触发 limit+1 分页
	stub := &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{
		friendRow("f1", now), friendRow("f2", now.Add(-time.Hour)), friendRow("f3", now.Add(-2*time.Hour)),
	}}}}
	page, err := newUserListRepo(stub).ListFriends(ctx, "u1", 2, "")
	if err != nil || len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("list friends page: %#v err=%v", page, err)
	}
	if page.Items[0].User.PresenceStatus == nil || page.Items[0].User.LastOnline == nil || page.Items[0].FollowedBackAt == nil {
		t.Fatalf("friend item fields: %#v", page.Items[0])
	}

	// 无分页（1 行）
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{friendRow("f1", now)}}}}
	page, err = newUserListRepo(stub).ListFriends(ctx, "u1", 2, "")
	if err != nil || len(page.Items) != 1 || page.NextCursor != nil {
		t.Fatalf("list friends one: %#v err=%v", page, err)
	}

	// 无效 cursor
	if _, err := newUserListRepo(&userListStub{}).ListFriends(ctx, "u1", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid cursor: %v", err)
	}
	// Query 错误
	stub = &userListStub{queryErr: errors.New("boom")}
	if _, err := newUserListRepo(stub).ListFriends(ctx, "u1", 2, ""); err == nil {
		t.Fatal("query error must propagate")
	}
	// 行 Scan 错误
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{friendRow("f1", now)}, scanErr: errors.New("scan")}}}
	if _, err := newUserListRepo(stub).ListFriends(ctx, "u1", 2, ""); err == nil {
		t.Fatal("scan error must propagate")
	}
	// 行尾 Err
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{friendRow("f1", now)}, err: errors.New("tail")}}}
	if _, err := newUserListRepo(stub).ListFriends(ctx, "u1", 2, ""); err == nil {
		t.Fatal("rows err must propagate")
	}
	// 非 QueryableDB
	if _, err := (&PostgresRepository{factory: &persistence.Factory{}}).ListFriends(ctx, "u1", 2, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("non-queryable: %v", err)
	}
}

func TestListFollowingFollowersGroupsPostgres(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Round(time.Second)

	// following：3 行分页 + scan 错误
	stub := &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{
		followingRow("t1", now), followingRow("t2", now.Add(-time.Hour)), followingRow("t3", now.Add(-2*time.Hour)),
	}}}}
	page, err := newUserListRepo(stub).ListFollowing(ctx, "u1", 2, "")
	if err != nil || len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("list following: %#v err=%v", page, err)
	}
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{followingRow("t1", now)}, scanErr: errors.New("scan")}}}
	if _, err := newUserListRepo(stub).ListFollowing(ctx, "u1", 2, ""); err == nil {
		t.Fatal("following scan error must propagate")
	}
	if _, err := newUserListRepo(&userListStub{}).ListFollowing(ctx, "u1", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("following invalid cursor: %v", err)
	}

	// followers：单行 + Query 错误
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{followerRow("t1", now)}}}}
	fpage, err := newUserListRepo(stub).ListFollowers(ctx, "u1", 2, "")
	if err != nil || len(fpage.Items) != 1 {
		t.Fatalf("list followers: %#v err=%v", fpage, err)
	}
	stub = &userListStub{queryErr: errors.New("boom")}
	if _, err := newUserListRepo(stub).ListFollowers(ctx, "u1", 2, ""); err == nil {
		t.Fatal("followers query error must propagate")
	}
	if _, err := newUserListRepo(&userListStub{}).ListFollowers(ctx, "u1", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("followers invalid cursor: %v", err)
	}

	// groups：3 行分页 + 行尾 Err
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{
		groupRow("g1", now), groupRow("g2", now.Add(-time.Hour)), groupRow("g3", now.Add(-2*time.Hour)),
	}}}}
	gpage, err := newUserListRepo(stub).ListGroups(ctx, "u1", 2, "")
	if err != nil || len(gpage.Items) != 2 || gpage.NextCursor == nil || *gpage.NextCursor == "" {
		t.Fatalf("list groups: %#v err=%v", gpage, err)
	}
	if gpage.Items[0].Location.Point.Latitude != 1.5 || gpage.Items[0].JoinMethod == "" {
		t.Fatalf("group item fields: %#v", gpage.Items[0])
	}
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{groupRow("g1", now)}, err: errors.New("tail")}}}
	if _, err := newUserListRepo(stub).ListGroups(ctx, "u1", 2, ""); err == nil {
		t.Fatal("groups rows err must propagate")
	}
	if _, err := newUserListRepo(&userListStub{}).ListGroups(ctx, "u1", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("groups invalid cursor: %v", err)
	}
}

func TestGetSettingsBatchPostgres(t *testing.T) {
	ctx := context.Background()

	// 空列表 → 空 map，不触库
	repo := newUserListRepo(&userListStub{})
	result, err := repo.GetSettingsBatch(ctx, nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("empty batch: %#v err=%v", result, err)
	}

	// 正常多行
	row := []any{"u1", "public", true, true, true, true, true, true, true, "everyone", true, true, true, true, true}
	stub := &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{row}}}}
	result, err = newUserListRepo(stub).GetSettingsBatch(ctx, []string{"u1", "u2"})
	if err != nil || len(result) != 1 || result["u1"].PrivacyLevel != "public" || result["u1"].AllowPrivateChat != "everyone" {
		t.Fatalf("settings batch: %#v err=%v", result, err)
	}

	// Query 错误
	stub = &userListStub{queryErr: errors.New("boom")}
	if _, err := newUserListRepo(stub).GetSettingsBatch(ctx, []string{"u1"}); err == nil {
		t.Fatal("settings query error must propagate")
	}
	// Scan 错误
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{row}, scanErr: errors.New("scan")}}}
	if _, err := newUserListRepo(stub).GetSettingsBatch(ctx, []string{"u1"}); err == nil {
		t.Fatal("settings scan error must propagate")
	}
	// 行尾 Err
	stub = &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{row}, err: errors.New("tail")}}}
	if _, err := newUserListRepo(stub).GetSettingsBatch(ctx, []string{"u1"}); err == nil {
		t.Fatal("settings rows err must propagate")
	}
}

func TestRelationshipBatchPostgres(t *testing.T) {
	ctx := context.Background()

	// 空列表 → 空 map
	repo := newUserListRepo(&userListStub{})
	result, err := repo.RelationshipBatch(ctx, "me", nil)
	if err != nil || len(result) != 0 {
		t.Fatalf("empty batch: %#v err=%v", result, err)
	}

	// 正常：forward 2 行 + reverse 2 行
	forward := &listRowsValues{rows: [][]any{
		{"t1", true, false}, {"t2", false, true},
	}}
	reverse := &listRowsValues{rows: [][]any{
		{"t1", true}, {"t2", false},
	}}
	stub := &userListStub{rowsSeq: []persistence.Rows{forward, reverse}}
	result, err = newUserListRepo(stub).RelationshipBatch(ctx, "me", []string{"t1", "t2"})
	if err != nil || len(result) != 2 {
		t.Fatalf("batch: %#v err=%v", result, err)
	}
	if !result["t1"].IsFollowing || !result["t1"].IsFollowedBy || result["t1"].IsBlocked {
		t.Fatalf("t1 rel: %#v", result["t1"])
	}
	if result["t2"].IsFollowing || !result["t2"].IsBlocked || result["t2"].IsFollowedBy {
		t.Fatalf("t2 rel: %#v", result["t2"])
	}

	cases := []struct {
		name string
		stub *userListStub
	}{
		{"forward query err", &userListStub{queryErr: errors.New("boom")}},
		{"forward scan err", &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{{"t1", true, false}}, scanErr: errors.New("scan")}}}},
		{"forward rows err", &userListStub{rowsSeq: []persistence.Rows{&listRowsValues{rows: [][]any{{"t1", true, false}}, err: errors.New("tail")}}}},
		{"reverse scan err", &userListStub{rowsSeq: []persistence.Rows{
			&listRowsValues{rows: [][]any{{"t1", true, false}}},
			&listRowsValues{rows: [][]any{{"t1", true}}, scanErr: errors.New("scan")},
		}}},
		{"reverse rows err", &userListStub{rowsSeq: []persistence.Rows{
			&listRowsValues{rows: [][]any{{"t1", true, false}}},
			&listRowsValues{rows: [][]any{{"t1", true}}, err: errors.New("tail")},
		}}},
	}
	for _, tc := range cases {
		if _, err := newUserListRepo(tc.stub).RelationshipBatch(ctx, "me", []string{"t1"}); err == nil {
			t.Fatalf("%s must fail", tc.name)
		}
	}
}

func TestCommonGroupCountAndUserExistsPostgres(t *testing.T) {
	ctx := context.Background()

	repo := newUserListRepo(&userListStub{row: userRelRow{values: []any{3}}})
	count, err := repo.CommonGroupCount(ctx, "u1", "u2")
	if err != nil || count != 3 {
		t.Fatalf("common count: %d err=%v", count, err)
	}
	// Scan 错误 → 传播
	repo = newUserListRepo(&userListStub{row: userRelRow{values: nil}})
	if _, err := repo.CommonGroupCount(ctx, "u1", "u2"); err == nil {
		t.Fatal("common count scan error must propagate")
	}

	repo = newUserListRepo(&userListStub{row: userRelRow{values: []any{1}}})
	exists, err := repo.UserExists(ctx, "u1")
	if err != nil || !exists {
		t.Fatalf("user exists: %v err=%v", exists, err)
	}
	// 查无此行 → false, nil
	repo = newUserListRepo(&userListStub{row: userRelRow{values: nil}})
	exists, err = repo.UserExists(ctx, "u1")
	if err != nil || exists {
		t.Fatalf("user missing: %v err=%v", exists, err)
	}
}

func TestScanRowsPostgresErrorPaths(t *testing.T) {
	scanErr := errors.New("scan")
	if _, _, err := scanFriendRow(testdb.RowValues{Err: scanErr}); err == nil {
		t.Fatal("scanFriendRow error must propagate")
	}
	if _, _, err := scanFollowingRow(testdb.RowValues{Err: scanErr}); err == nil {
		t.Fatal("scanFollowingRow error must propagate")
	}
	if _, _, err := scanFollowerRow(testdb.RowValues{Err: scanErr}); err == nil {
		t.Fatal("scanFollowerRow error must propagate")
	}
	if _, _, err := scanUserGroupRow(testdb.RowValues{Err: scanErr}); err == nil {
		t.Fatal("scanUserGroupRow error must propagate")
	}
}
