package groups

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// groupFullStub 按 SQL 常量分派 QueryRow/Query/Exec，覆盖 repo_postgres.go
// 各函数的正常与错误路径（现有 groupBackendStub 只支持 Create/Get/Search）。
type groupFullStub struct {
	queryRows       map[string][][]any
	queryRowsErr    map[string]error
	queryCustomRows map[string]persistence.Rows
	queryRow        map[string][]any
	queryRowErr     map[string]error
	execErrs        map[string]error
}

func (b *groupFullStub) Ping(context.Context) error { return nil }
func (b *groupFullStub) Close() error               { return nil }
func (b *groupFullStub) Exec(_ context.Context, query string, _ ...any) error {
	return b.execErrs[query]
}
func (b *groupFullStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if err := b.queryRowErr[query]; err != nil {
		return testdb.RowValues{Err: err}
	}
	if values, ok := b.queryRow[query]; ok {
		return testdb.RowValues{Values: values}
	}
	return testdb.RowValues{Err: persistence.ErrNotImplemented}
}
func (b *groupFullStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	// 匹配顺序：先精确（queryRowsErr/queryCustomRows/queryRows 的 map key），
	// 再前缀兜底。前缀匹配是必要的——ListMyGroups 等查询在常量前缀后动态
	// 拼接 keyset 谓词与 LIMIT（见 queryListMyGroupsBase），无法精确命中；
	// 前缀 key 因此必须是「执行 SQL 的前缀」而非任意子串，防止掩盖漂移。
	for key, err := range b.queryRowsErr {
		if query == key || strings.HasPrefix(query, key) {
			return nil, err
		}
	}
	if rows, ok := b.queryCustomRows[query]; ok {
		return rows, nil
	}
	for key, rows := range b.queryCustomRows {
		if strings.HasPrefix(query, key) {
			return rows, nil
		}
	}
	for key, rows := range b.queryRows {
		if query == key || (strings.HasPrefix(query, key)) {
			return &testdb.RowsValues{Rows: rows}, nil
		}
	}
	return &persistence.InMemoryRows{}, nil
}

func newGroupsFullRepo(stub *groupFullStub) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = stub
	return NewPostgresRepository(factory)
}

var (
	gTestTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gTestNow  = gTestTime.Add(time.Hour)
)

// 18 列，与 queryGetGroup/scanGroupBase 对应；requiresApproval 由第 10 列控制。
// 注意 groupRowValues 的 **time.Time/**int/**string case 接受裸值。
func groupBaseRow(requiresApproval bool) []any {
	// 列序对齐 queryGetGroup：... is_searchable, members_searchable, max_members, ...
	return []any{"group_1", "群", "", "描述", false, "位置", 1.5, 2.5, "direct", requiresApproval, nil, true, true, 200, "owner", gTestTime, gTestTime, []string{"tag"}, 1}
}

func TestPostgresUpdateDismissAndExecWrappers(t *testing.T) {
	ctx := context.Background()

	// Update：全字段路径（清密码 + 位置）
	stub := &groupFullStub{}
	repo := newGroupsFullRepo(stub)
	clear := true
	joinMethod := GroupJoinMethodInviteOnly
	searchable := false
	maxMembers := 50
	err := repo.Update(ctx, "u1", UpdateGroupRequest{
		GroupID: "group_1", Name: "新名", ClearPassword: &clear, JoinMethod: &joinMethod,
		RequiresApproval: &clear, IsSearchable: &searchable, MaxMembers: &maxMembers,
		Location: &Location{LocationName: "新位置", Point: GeoPoint{Latitude: 2, Longitude: 3}},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	// 密码字段路径
	pwd := "secret"
	if err := repo.Update(ctx, "u1", UpdateGroupRequest{GroupID: "group_1", Password: &pwd}); err != nil {
		t.Fatalf("update pwd: %v", err)
	}
	// Exec 错误传播
	stub = &groupFullStub{execErrs: map[string]error{queryUpdateGroup: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).Update(ctx, "u1", UpdateGroupRequest{GroupID: "group_1", Name: "x"}); err == nil {
		t.Fatal("update exec error must propagate")
	}
	// 未实现路径
	if err := (&PostgresRepository{}).Update(ctx, "u1", UpdateGroupRequest{GroupID: "g"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("update nil factory: %v", err)
	}

	// Dismiss
	if err := newGroupsFullRepo(&groupFullStub{}).Dismiss(ctx, "group_1"); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	stub = &groupFullStub{execErrs: map[string]error{queryDeleteGroup: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).Dismiss(ctx, "group_1"); err == nil {
		t.Fatal("dismiss exec error must propagate")
	}

	// 纯 Exec 转发：SetMemberRole / MuteMember / UnmuteMember / BanMember /
	// UnbanMember / RemoveMember / RejectApplication / UpdatePreferences /
	// LeaveGroup / CancelApplication / RejectInvite / RevokeInvite /
	// DeleteAnnouncement
	execCases := []struct {
		name string
		call func(repo *PostgresRepository) error
	}{
		{"set role", func(r *PostgresRepository) error {
			return r.SetMemberRole(ctx, "u1", SetMemberRoleRequest{GroupID: "g", UserID: "u2", Role: GroupRoleAdmin})
		}},
		{"mute", func(r *PostgresRepository) error {
			return r.MuteMember(ctx, "u1", MuteMemberRequest{GroupID: "g", UserID: "u2", DurationMinutes: 60})
		}},
		{"unmute", func(r *PostgresRepository) error {
			return r.UnmuteMember(ctx, "u1", UnmuteMemberRequest{GroupID: "g", UserID: "u2"})
		}},
		{"ban", func(r *PostgresRepository) error {
			return r.BanMember(ctx, "u1", BanMemberRequest{GroupID: "g", UserID: "u2"})
		}},
		{"unban", func(r *PostgresRepository) error {
			return r.UnbanMember(ctx, "u1", UnbanMemberRequest{GroupID: "g", UserID: "u2"})
		}},
		{"remove", func(r *PostgresRepository) error {
			return r.RemoveMember(ctx, "u1", RemoveMemberRequest{GroupID: "g", UserID: "u2"})
		}},
		{"reject app", func(r *PostgresRepository) error { return r.RejectApplication(ctx, "a1", nil) }},
		{"my settings", func(r *PostgresRepository) error {
			return r.UpdatePreferences(ctx, "u1", UpdateGroupPreferencesRequest{GroupID: "g"})
		}},
		{"leave", func(r *PostgresRepository) error { return r.LeaveGroup(ctx, "u1", "g") }},
		{"cancel app", func(r *PostgresRepository) error { return r.CancelApplication(ctx, "a1") }},
		{"reject invite", func(r *PostgresRepository) error { return r.RejectInvite(ctx, "i1") }},
		{"revoke invite", func(r *PostgresRepository) error { return r.RevokeInvite(ctx, "g", "i1") }},
		{"delete ann", func(r *PostgresRepository) error { return r.DeleteAnnouncement(ctx, "g", "a1") }},
	}
	for _, tc := range execCases {
		if err := tc.call(newGroupsFullRepo(&groupFullStub{})); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if err := tc.call(&PostgresRepository{}); !errors.Is(err, persistence.ErrNotImplemented) {
			t.Fatalf("%s nil factory: %v", tc.name, err)
		}
	}
	// ban 的插入失败路径
	stub = &groupFullStub{execErrs: map[string]error{queryInsertGroupBan: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).BanMember(ctx, "u1", BanMemberRequest{GroupID: "g", UserID: "u2"}); err == nil {
		t.Fatal("ban insert error must propagate")
	}
}

func TestPostgresListMembersAndSearchMembers(t *testing.T) {
	ctx := context.Background()
	memberRow := func(uid string, role string) []any {
		return []any{uid, "昵称", role, gTestTime}
	}
	stub := &groupFullStub{queryRows: map[string][][]any{
		queryListGroupMembers:   {memberRow("u1", "owner"), memberRow("u2", "member")},
		querySearchGroupMembers: {memberRow("u3", "admin")},
	}}
	repo := newGroupsFullRepo(stub)
	role := GroupRoleAdmin
	items, _, err := repo.ListMembers(ctx, "group_1", &role, 20, "")
	if err != nil || len(items) != 2 || items[0].Role != GroupRoleOwner || items[0].JoinedAt == "" {
		t.Fatalf("list members: %#v err=%v", items, err)
	}
	items, _, err = repo.SearchMembers(ctx, "group_1", "kw", nil, 20, "")
	if err != nil || len(items) != 1 {
		t.Fatalf("search members: %#v err=%v", items, err)
	}
	// Query 错误
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListGroupMembers: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListMembers(ctx, "g", nil, 20, ""); err == nil {
		t.Fatal("list members query error must propagate")
	}
	// rows.Err 尾错误
	stub = &groupFullStub{queryCustomRows: map[string]persistence.Rows{
		queryListGroupMembers: &groupRowsErrValues{rows: [][]any{{"u1", "n", "member", gTestTime}}, err: errors.New("tail")},
	}}
	if _, _, err := newGroupsFullRepo(stub).ListMembers(ctx, "g", nil, 20, ""); err == nil {
		t.Fatal("list members tail error must propagate")
	}
	// SearchMembers 的 Query 错误
	stubErr := &groupFullStub{queryRowsErr: map[string]error{querySearchGroupMembers: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stubErr).SearchMembers(ctx, "g", "", nil, 20, ""); err == nil {
		t.Fatal("search members query error must propagate")
	}
	// 未实现
	if _, _, err := (&PostgresRepository{}).ListMembers(ctx, "g", nil, 20, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("list members nil factory: %v", err)
	}
}

func TestPostgresIsMemberAndGetApplication(t *testing.T) {
	ctx := context.Background()

	// 命中 → true
	stub := &groupFullStub{queryRow: map[string][]any{queryGetGroupMembership: {"member", gTestTime, false, nil}}}
	ok, err := newGroupsFullRepo(stub).IsMember(ctx, "g", "u1")
	if err != nil || !ok {
		t.Fatalf("is member: %v err=%v", ok, err)
	}
	// 无行 → false, nil
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows}}
	ok, err = newGroupsFullRepo(stub).IsMember(ctx, "g", "u1")
	if err != nil || ok {
		t.Fatalf("is member missing: %v err=%v", ok, err)
	}
	// 其他错误传播
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupMembership: errors.New("boom")}}
	if _, err := newGroupsFullRepo(stub).IsMember(ctx, "g", "u1"); err == nil {
		t.Fatal("is member error must propagate")
	}

	// GetApplication：存在（loadApplication 需要 queryGetGroupApplication + queryGetGroup）
	appRow := []any{"a1", "group_1", "u2", "备注", "pending", gTestTime, nil, nil}
	stub = &groupFullStub{
		queryRow: map[string][]any{queryGetGroupApplication: appRow, queryGetGroup: groupBaseRow(false)},
	}
	item, found, err := newGroupsFullRepo(stub).GetApplication(ctx, "a1")
	if err != nil || !found || item.ApplicationID != "a1" || item.GroupID != "group_1" {
		t.Fatalf("get application: %#v found=%v err=%v", item, found, err)
	}
	// 不存在 → false, nil
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupApplication: pgx.ErrNoRows}}
	if _, found, err := newGroupsFullRepo(stub).GetApplication(ctx, "a1"); err != nil || found {
		t.Fatalf("get application missing: found=%v err=%v", found, err)
	}
	// 其他错误传播
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupApplication: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).GetApplication(ctx, "a1"); err == nil {
		t.Fatal("get application error must propagate")
	}
}

func TestPostgresListMyGroupsPaginationAndFields(t *testing.T) {
	ctx := context.Background()
	myGroupRow := func(gid string, hasLastMessage bool, isPinned bool, unreadCount int) []any {
		row := []any{gid, "群" + gid, "", "描述", false, "位置", 1.5, 2.5, "direct", false, nil, true, 200, "owner", gTestTime, gTestTime, []string{}, 3, 1, "member", gTestTime, false, nil, isPinned, unreadCount}
		if hasLastMessage {
			row = append(row, "m1", "text", "内容", "u9", "发件人", "", gTestNow)
		} else {
			row = append(row, nil, nil, nil, nil, nil, nil, nil)
		}
		return row
	}
	// 3 行 + limit=2 → 分页
	stub := &groupFullStub{queryRows: map[string][][]any{queryListMyGroupsBase: {
		myGroupRow("g1", true, true, 5), myGroupRow("g2", false, false, 0), myGroupRow("g3", true, false, 0),
	}}}
	items, next, err := newGroupsFullRepo(stub).ListMyGroups(ctx, "u1", 2, "")
	if err != nil || len(items) != 2 || next == "" {
		t.Fatalf("list my groups: %#v next=%q err=%v", items, next, err)
	}
	if items[0].LastMessage == nil || items[0].LastMessage.MessageID != "m1" || items[0].ViewerState == nil || items[0].OnlineMembersCount == nil {
		t.Fatalf("my groups fields: %#v", items[0])
	}
	if items[0].ViewerState.IsPinned != true || items[0].ViewerState.UnreadCount != 5 {
		t.Fatalf("my groups member state: %#v", items[0].ViewerState)
	}
	// 带 pageToken 的 keyset 路径
	token, err := cursor.Encode(myGroupsCursorNamespace, myGroupsCursor{JoinedAt: gTestTime.Format(time.RFC3339Nano), GroupID: "g1"})
	if err != nil {
		t.Fatalf("encode token: %v", err)
	}
	stub = &groupFullStub{queryRows: map[string][][]any{queryListMyGroupsBase: {myGroupRow("g1", false, false, 0)}}}
	if _, _, err := newGroupsFullRepo(stub).ListMyGroups(ctx, "u1", 2, token); err != nil {
		t.Fatalf("list my groups token: %v", err)
	}
	// 无效 token
	if _, _, err := newGroupsFullRepo(&groupFullStub{}).ListMyGroups(ctx, "u1", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("invalid token: %v", err)
	}
	// limit 归一化
	if _, _, err := newGroupsFullRepo(&groupFullStub{}).ListMyGroups(ctx, "u1", 0, ""); err != nil {
		t.Fatalf("limit zero: %v", err)
	}
	// Query 错误
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListMyGroupsBase: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListMyGroups(ctx, "u1", 2, ""); err == nil {
		t.Fatal("my groups query error must propagate")
	}
}

func TestPostgresJoinGroupBranches(t *testing.T) {
	ctx := context.Background()
	directRow := groupBaseRow(false)
	approvalRow := groupBaseRow(true)

	// direct：非成员 → 插入成员
	stub := &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: directRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows},
	}
	res, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil)
	if err != nil || res.Status != EnterGroupStatusJoined {
		t.Fatalf("join direct: %#v err=%v", res, err)
	}
	// 已加入 → 直接返回 joined
	stub = &groupFullStub{
		queryRow: map[string][]any{queryGetGroup: directRow, queryGetGroupMembership: {"member", gTestTime, false, nil}},
	}
	res, err = newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil)
	if err != nil || res.Status != EnterGroupStatusJoined {
		t.Fatalf("join already: %#v err=%v", res, err)
	}
	// 群不存在
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroup: pgx.ErrNoRows}}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "g", nil, nil); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("join missing group: %v", err)
	}
	// getGroup 其他错误
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroup: errors.New("boom")}}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "g", nil, nil); err == nil {
		t.Fatal("join get group error must propagate")
	}
	// membership 查询错误
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: directRow},
		queryRowErr: map[string]error{queryGetGroupMembership: errors.New("boom")},
	}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "g", nil, nil); err == nil {
		t.Fatal("join membership error must propagate")
	}
	// password：缺密码 → ErrInvalidPassword
	inviteRow := directRow
	inviteRow[8] = "password"
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: inviteRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows},
	}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("join password missing: %v", err)
	}
	// invite_only：无 pending 邀请 → ErrPermissionDenied
	inviteRow = directRow
	inviteRow[8] = "invite_only"
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: inviteRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows, queryFindPendingGroupInvite: pgx.ErrNoRows},
	}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("join invite-only: %v", err)
	}
	// requires_approval：命中已有 pending 申请 → 返回现有
	pwd := "p"
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: approvalRow, queryFindPendingGroupApplication: {"app_exist"}},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows},
	}
	res, err = newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", &pwd, nil)
	if err != nil || res.Status != EnterGroupStatusPendingApproval || res.ApplicationID == nil || *res.ApplicationID != "app_exist" || res.IsNewApplication {
		t.Fatalf("join approval existing: %#v err=%v", res, err)
	}
	// requires_approval：无申请 → 新建
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: approvalRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows, queryFindPendingGroupApplication: pgx.ErrNoRows},
	}
	res, err = newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, &pwd)
	if err != nil || res.Status != EnterGroupStatusPendingApproval || !res.IsNewApplication {
		t.Fatalf("join approval new: %#v err=%v", res, err)
	}
	// requires_approval：插入申请失败
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: approvalRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows, queryFindPendingGroupApplication: pgx.ErrNoRows},
		execErrs:    map[string]error{queryInsertGroupApplication: errors.New("boom")},
	}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil); err == nil {
		t.Fatal("join approval insert error must propagate")
	}
	// direct：插入成员失败
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroup: directRow},
		queryRowErr: map[string]error{queryGetGroupMembership: pgx.ErrNoRows},
		execErrs:    map[string]error{queryInsertGroupMember: errors.New("boom")},
	}
	if _, err := newGroupsFullRepo(stub).JoinGroup(ctx, "u2", "group_1", nil, nil); err == nil {
		t.Fatal("join insert member error must propagate")
	}
}

func TestPostgresAnnouncementLifecycle(t *testing.T) {
	ctx := context.Background()
	annRow := func(id string) []any {
		return []any{id, "group_1", "标题", "内容", "u1", true, gTestTime}
	}

	// Create
	stub := &groupFullStub{}
	item, err := newGroupsFullRepo(stub).CreateAnnouncement(ctx, "u1", CreateGroupAnnouncementRequest{GroupID: "group_1", Title: "标题", Content: "内容"})
	if err != nil || item.AnnouncementID == "" {
		t.Fatalf("create announcement: %#v err=%v", item, err)
	}
	// exec 失败
	stub = &groupFullStub{execErrs: map[string]error{queryInsertGroupAnnouncement: errors.New("boom")}}
	if _, err := newGroupsFullRepo(stub).CreateAnnouncement(ctx, "u1", CreateGroupAnnouncementRequest{GroupID: "g"}); err == nil {
		t.Fatal("create announcement exec error must propagate")
	}

	// List：2 行
	stub = &groupFullStub{queryRows: map[string][][]any{queryListGroupAnnouncements: {annRow("a1"), annRow("a2")}}}
	items, err := newGroupsFullRepo(stub).ListAnnouncements(ctx, "group_1")
	if err != nil || len(items) != 2 || !items[0].IsPinned || items[0].CreatedAt == "" {
		t.Fatalf("list announcements: %#v err=%v", items, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListGroupAnnouncements: errors.New("boom")}}
	if _, err := newGroupsFullRepo(stub).ListAnnouncements(ctx, "group_1"); err == nil {
		t.Fatal("list announcements query error must propagate")
	}

	// Update（全字段 + nil 字段）
	title := "新标题"
	stub = &groupFullStub{}
	if err := newGroupsFullRepo(stub).UpdateAnnouncement(ctx, "group_1", UpdateGroupAnnouncementRequest{AnnouncementID: "a1", Title: &title}); err != nil {
		t.Fatalf("update announcement: %v", err)
	}
	stub = &groupFullStub{execErrs: map[string]error{queryUpdateGroupAnnouncement: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).UpdateAnnouncement(ctx, "g", UpdateGroupAnnouncementRequest{AnnouncementID: "a1"}); err == nil {
		t.Fatal("update announcement exec error must propagate")
	}
}

func TestPostgresApplicationsAndInvitesLists(t *testing.T) {
	ctx := context.Background()
	appRow := func(id string) []any {
		return []any{id, "u2", "备注", "pending", gTestTime, gTestTime, "u1"}
	}
	myAppRow := func(id string) []any {
		return []any{id, "group_1", "备注", "pending", gTestTime, nil, nil}
	}
	inviteRow := func(id string) []any {
		return []any{id, "group_1", "群名", "", 3, "u1", "消息", "pending", gTestTime, gTestTime, nil}
	}
	sentRow := func(id string) []any {
		return []any{id, "u2", "消息", "pending", gTestTime, gTestTime, gTestTime}
	}

	status := ApplicationStatusPending
	// ListApplications
	stub := &groupFullStub{queryRows: map[string][][]any{queryListGroupApplications: {appRow("a1"), appRow("a2")}}}
	items, _, err := newGroupsFullRepo(stub).ListApplications(ctx, "group_1", &status, 20, "")
	if err != nil || len(items) != 2 || items[0].Message == nil || items[0].HandledAt == nil {
		t.Fatalf("list applications: %#v err=%v", items, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListGroupApplications: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListApplications(ctx, "g", nil, 20, ""); err == nil {
		t.Fatal("list applications query error must propagate")
	}

	// ListMyApplications
	stub = &groupFullStub{queryRows: map[string][][]any{queryListMyGroupApplications: {myAppRow("a1")}}}
	items, _, err = newGroupsFullRepo(stub).ListMyApplications(ctx, "u2", nil, 20, "")
	if err != nil || len(items) != 1 || items[0].Message == nil {
		t.Fatalf("list my applications: %#v err=%v", items, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListMyGroupApplications: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListMyApplications(ctx, "u2", nil, 20, ""); err == nil {
		t.Fatal("list my applications query error must propagate")
	}

	// ListMyInvites
	stub = &groupFullStub{queryRows: map[string][][]any{queryListMyInvites: {inviteRow("i1")}}}
	invites, _, err := newGroupsFullRepo(stub).ListMyInvites(ctx, "u2", nil, 20, "")
	if err != nil || len(invites) != 1 || invites[0].ExpiresAt == nil || invites[0].Message == nil {
		t.Fatalf("list my invites: %#v err=%v", invites, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListMyInvites: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListMyInvites(ctx, "u2", nil, 20, ""); err == nil {
		t.Fatal("list my invites query error must propagate")
	}

	// ListSentInvites
	stub = &groupFullStub{queryRows: map[string][][]any{queryListSentInvites: {sentRow("i1")}}}
	sent, _, err := newGroupsFullRepo(stub).ListSentInvites(ctx, "group_1", &status, 20, "")
	if err != nil || len(sent) != 1 || sent[0].ExpiresAt == nil || sent[0].RespondedAt == nil {
		t.Fatalf("list sent invites: %#v err=%v", sent, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListSentInvites: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListSentInvites(ctx, "group_1", nil, 20, ""); err == nil {
		t.Fatal("list sent invites query error must propagate")
	}

	// ListBannedMembers
	banRow := func(uid string) []any {
		return []any{uid, gTestTime, "理由", "u1"}
	}
	stub = &groupFullStub{queryRows: map[string][][]any{queryListGroupBans: {banRow("u2"), banRow("u3")}}}
	banned, _, err := newGroupsFullRepo(stub).ListBannedMembers(ctx, "group_1", 20, "")
	if err != nil || len(banned) != 2 || banned[0].Reason == nil || banned[0].BannedBy == nil || banned[0].BannedAt == "" {
		t.Fatalf("list banned: %#v err=%v", banned, err)
	}
	stub = &groupFullStub{queryRowsErr: map[string]error{queryListGroupBans: errors.New("boom")}}
	if _, _, err := newGroupsFullRepo(stub).ListBannedMembers(ctx, "g", 20, ""); err == nil {
		t.Fatal("list banned query error must propagate")
	}
}

func strPtr(s string) *string { return &s }

func TestPostgresApproveAcceptTxAndInviteMember(t *testing.T) {
	ctx := context.Background()
	appRow := []any{"a1", "group_1", "u2", "", "pending", gTestTime, nil, nil}
	inviteRow := []any{"i1", "group_1", "u1", "u2", "", "pending", gTestTime, gTestTime, nil}

	// ApproveApplication：loadApplication + tx 内两个 Exec
	stub := &groupFullStub{
		queryRow: map[string][]any{queryGetGroupApplication: appRow, queryGetGroup: groupBaseRow(false)},
	}
	if err := newGroupsFullRepo(stub).ApproveApplication(ctx, "u1", "a1"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	// loadApplication 失败传播
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupApplication: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).ApproveApplication(ctx, "u1", "a1"); err == nil {
		t.Fatal("approve load error must propagate")
	}

	// AcceptInvite：loadInvite + tx 内两个 Exec
	stub = &groupFullStub{
		queryRow: map[string][]any{
			queryGetGroupInvite: inviteRow, queryGetGroup: groupBaseRow(false), queryCountGroupMembers: {3},
		},
	}
	if err := newGroupsFullRepo(stub).AcceptInvite(ctx, "u2", "i1"); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	// loadInvite 失败
	stub = &groupFullStub{queryRowErr: map[string]error{queryGetGroupInvite: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).AcceptInvite(ctx, "u2", "i1"); err == nil {
		t.Fatal("accept invite load error must propagate")
	}
	// loadInvite 中 group 查询失败
	stub = &groupFullStub{
		queryRow:    map[string][]any{queryGetGroupInvite: inviteRow},
		queryRowErr: map[string]error{queryGetGroup: errors.New("boom")},
	}
	if err := newGroupsFullRepo(stub).AcceptInvite(ctx, "u2", "i1"); err == nil {
		t.Fatal("accept invite group error must propagate")
	}

	// InviteMember：已有 pending → 返回现有
	stub = &groupFullStub{queryRow: map[string][]any{queryFindPendingGroupInvite: {"i_exist", gTestTime}}}
	inv, err := newGroupsFullRepo(stub).InviteMember(ctx, "u1", InviteMemberRequest{GroupID: "group_1", UserID: "u2"}, nil)
	if err != nil || inv.InviteID != "i_exist" {
		t.Fatalf("invite existing: %#v err=%v", inv, err)
	}
	// 新建（TTL=0 永不过期 → expires_at 省略）
	stub = &groupFullStub{queryRowErr: map[string]error{queryFindPendingGroupInvite: pgx.ErrNoRows}}
	inv, err = newGroupsFullRepo(stub).InviteMember(ctx, "u1", InviteMemberRequest{GroupID: "group_1", UserID: "u2", Message: strPtr("来")}, nil)
	if err != nil || inv.InviteID == "" || inv.ExpiresAt != "" {
		t.Fatalf("invite new (nil ttl): %#v err=%v", inv, err)
	}
	// 新建（有 TTL → expires_at 返回）
	exp := time.Now().UTC().Add(72 * time.Hour)
	stub = &groupFullStub{queryRowErr: map[string]error{queryFindPendingGroupInvite: pgx.ErrNoRows}}
	inv, err = newGroupsFullRepo(stub).InviteMember(ctx, "u1", InviteMemberRequest{GroupID: "group_1", UserID: "u2"}, &exp)
	if err != nil || inv.InviteID == "" || inv.ExpiresAt == "" {
		t.Fatalf("invite new (with ttl): %#v err=%v", inv, err)
	}
	// 插入失败
	stub = &groupFullStub{
		queryRowErr: map[string]error{queryFindPendingGroupInvite: pgx.ErrNoRows},
		execErrs:    map[string]error{queryInsertGroupInvite: errors.New("boom")},
	}
	if _, err := newGroupsFullRepo(stub).InviteMember(ctx, "u1", InviteMemberRequest{GroupID: "g", UserID: "u2"}, nil); err == nil {
		t.Fatal("invite insert error must propagate")
	}
}

func TestPostgresTransferOwnershipAndNilHelpers(t *testing.T) {
	ctx := context.Background()
	repo := newGroupsFullRepo(&groupFullStub{})
	if err := repo.TransferOwnership(ctx, "u1", TransferOwnershipRequest{GroupID: "g", NewOwnerID: "u2"}); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	stub := &groupFullStub{execErrs: map[string]error{queryUpdateGroupMemberRole: errors.New("boom")}}
	if err := newGroupsFullRepo(stub).TransferOwnership(ctx, "u1", TransferOwnershipRequest{GroupID: "g", NewOwnerID: "u2"}); err == nil {
		t.Fatal("transfer exec error must propagate")
	}

	// nil helper 函数
	if nilOrString("") != nil || nilOrString("x") == nil {
		t.Fatal("nilOrString")
	}
	jm := GroupJoinMethodInviteOnly
	if nilOrStringJoinMethod(nil) != nil || nilOrStringJoinMethod(&jm) == nil {
		t.Fatal("nilOrStringJoinMethod")
	}
	b := true
	if nilOrBool(&b) != &b {
		t.Fatal("nilOrBool")
	}
	i := 5
	if nilOrInt(&i) != &i {
		t.Fatal("nilOrInt")
	}
	if nilOrStrings(nil) != nil || len(nilOrStrings([]string{"a"})) != 1 {
		t.Fatal("nilOrStrings")
	}
	if derefStringPtr(nil) != "" || derefStringPtr(&jm2String) != jm2String {
		t.Fatal("derefStringPtr")
	}
	if derefTime(nil) != "" || derefTime(&gTestTime) == "" {
		t.Fatal("derefTime")
	}
	if nullableStringPtr(nil) != nil || nullableStringPtr(strPtr("")) != nil || nullableStringPtr(strPtr("v")) != "v" {
		t.Fatal("nullableStringPtr")
	}

	// queryableForTx
	ctxWithQ, cancel := context.WithCancel(ctx)
	_ = cancel
	_, ok := persistence.QueryableForTx(ctxWithQ, &persistence.Factory{DB: &groupFullStub{}})
	if !ok {
		t.Fatal("queryableForTx with db must succeed")
	}
	if _, ok := persistence.QueryableForTx(ctx, nil); ok {
		t.Fatal("queryableForTx nil factory must fail")
	}
	if _, ok := persistence.QueryableForTx(ctx, &persistence.Factory{}); ok {
		t.Fatal("queryableForTx empty factory must fail")
	}
	// ctx 带 transaction queryable 优先
	q := &groupFullStub{}
	txCtx := persistence.WithTx(ctx, q, nil)
	if got, ok := persistence.QueryableForTx(txCtx, nil); !ok || got != q {
		t.Fatal("queryableForTx must prefer ctx queryable")
	}
}

var jm2String = "invite_only"

func TestPostgresRowsErrPaths(t *testing.T) {
	ctx := context.Background()
	// 32 列 my-groups 行（含 is_pinned、unread_count）
	myRow := []any{"g1", "群", "", "d", false, "", 0.0, 0.0, "direct", false, nil, true, 10, "o", nil, nil, []string{}, 1, 0, "member", gTestTime, false, nil, false, 0, nil, nil, nil, nil, nil, nil, nil}
	tailStub := &groupFullStub{queryCustomRows: map[string]persistence.Rows{
		queryListMyGroupsBase: &groupRowsErrValues{rows: [][]any{myRow}, err: errors.New("tail")},
	}}
	if _, _, err := newGroupsFullRepo(tailStub).ListMyGroups(ctx, "u1", 2, ""); err == nil {
		t.Fatal("my groups tail error must propagate")
	}
	// ListAnnouncements / ListApplications / ListMyApplications / ListMyInvites /
	// ListSentInvites / ListBannedMembers 的 rows.Err 尾错误
	rowsErrCases := []struct {
		key  string
		row  []any
		call func(repo *PostgresRepository) error
	}{
		{queryListGroupAnnouncements, []any{"a1", "g", "t", "c", "u1", false, gTestTime},
			func(r *PostgresRepository) error { _, err := r.ListAnnouncements(ctx, "g"); return err }},
		{queryListGroupApplications, []any{"a1", "u2", "", "pending", gTestTime, nil, nil},
			func(r *PostgresRepository) error { _, _, err := r.ListApplications(ctx, "g", nil, 20, ""); return err }},
		{queryListMyGroupApplications, []any{"a1", "g", "", "pending", gTestTime, nil, nil},
			func(r *PostgresRepository) error {
				_, _, err := r.ListMyApplications(ctx, "u2", nil, 20, "")
				return err
			}},
		{queryListMyInvites, []any{"i1", "g", "群", "", 3, "u1", "", "pending", gTestTime, nil, nil},
			func(r *PostgresRepository) error { _, _, err := r.ListMyInvites(ctx, "u2", nil, 20, ""); return err }},
		{queryListSentInvites, []any{"i1", "u2", "", "pending", gTestTime, nil, nil},
			func(r *PostgresRepository) error { _, _, err := r.ListSentInvites(ctx, "g", nil, 20, ""); return err }},
		{queryListGroupBans, []any{"u2", gTestTime, nil, nil},
			func(r *PostgresRepository) error { _, _, err := r.ListBannedMembers(ctx, "g", 20, ""); return err }},
	}
	for _, tc := range rowsErrCases {
		stub := &groupFullStub{queryCustomRows: map[string]persistence.Rows{
			tc.key: &groupRowsErrValues{rows: [][]any{tc.row}, err: errors.New("tail")},
		}}
		if err := tc.call(newGroupsFullRepo(stub)); err == nil {
			t.Fatalf("%s tail error must propagate", tc.key)
		}
	}
	// 空列表正常路径
	stub := &groupFullStub{queryRows: map[string][][]any{queryListGroupBans: {}}}
	if _, _, err := newGroupsFullRepo(stub).ListBannedMembers(ctx, "g", 20, ""); err != nil {
		t.Fatalf("list banned empty: %v", err)
	}
}

// groupRowsErrValues 为 rows.Err() 非 nil 的 rows，用于覆盖列表函数的尾错误分支。
type groupRowsErrValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *groupRowsErrValues) Next() bool { return r.idx < len(r.rows) }
func (r *groupRowsErrValues) Scan(dest ...any) error {
	row := testdb.RowValues{Values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *groupRowsErrValues) Close() error { return nil }
func (r *groupRowsErrValues) Err() error   { return r.err }
