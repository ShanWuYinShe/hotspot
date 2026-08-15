package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// adminFullStub 按 SQL 常量精确分派，覆盖 admin repo 其余函数
// （audit/sessions/users/groups/reports/recall 等，现有 adminBackendStub 未覆盖）。
type adminFullStub struct {
	queryRows       map[string][][]any
	queryRowsErr    map[string]error
	queryCustomRows map[string]persistence.Rows
	queryRow        map[string][]any
	queryRowErr     map[string]error
	execErrs        map[string]error
}

func (b *adminFullStub) Ping(context.Context) error { return nil }
func (b *adminFullStub) Close() error               { return nil }
func (b *adminFullStub) Exec(_ context.Context, query string, _ ...any) error {
	return b.execErrs[query]
}
func (b *adminFullStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if err := b.queryRowErr[query]; err != nil {
		return adminRowValues{err: err}
	}
	if values, ok := b.queryRow[query]; ok {
		return adminRowValues{values: values}
	}
	return adminRowValues{err: persistence.ErrNotImplemented}
}
func (b *adminFullStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if err := b.queryRowsErr[query]; err != nil {
		return nil, err
	}
	if rows, ok := b.queryCustomRows[query]; ok {
		return rows, nil
	}
	if rows, ok := b.queryRows[query]; ok {
		return &adminRowsValues{rows: rows}, nil
	}
	return &persistence.InMemoryRows{}, nil
}

func newAdminFullRepo(stub *adminFullStub) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = stub
	return NewPostgresRepository(factory)
}

func newAdminFullStore(stub *adminFullStub) *PersistenceModerationStore {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = stub
	return NewPersistenceModerationStore(factory, nil)
}

var aTestTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func adminUserRowVals(uid string) []any {
	return []any{uid, "昵称" + uid, "av", "bio", "online", aTestTime, "位置", 1.5, 2.5, "normal", aTestTime, "normal", "login_" + uid, "ip1", "loc1", "ip2", "loc2", "e@x.com", "13800000000"}
}

func adminGroupRowVals17(gid string) []any {
	return []any{gid, "群" + gid, "", "描述", false, "位置", 1.5, 2.5, "direct", true, 200, aTestTime, "owner", "群主", "", 3, aTestTime}
}

func adminReportRowVals16(rid string) []any {
	return []any{rid, "moment", "m1", "理由", "描述", "pending", aTestTime, nil, "u9", "举报人", "", "", "", "", "", nil}
}

func adminAuditRowVals(logID string) []any {
	return []any{logID, "role.create", "op1", "操作者", "", "role", "role_1", "角色", `{"k":"v"}`, "ip1", "ip2", "device", aTestTime}
}

func TestAdminRepoListPermissionsAndNilHelpers(t *testing.T) {
	ctx := context.Background()
	repo := newAdminFullRepo(&adminFullStub{})
	perms, err := repo.ListPermissions(ctx)
	if err != nil || len(perms) == 0 {
		t.Fatalf("list permissions: %#v err=%v", perms, err)
	}
	if _, err := (&PostgresRepository{}).ListPermissions(ctx); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("list permissions nil factory: %v", err)
	}
	if nullableString("  ") != nil || nullableString("x") != "x" {
		t.Fatal("nullableString")
	}
	if !persistence.IsUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("isUniqueViolation must match 23505")
	}
	if persistence.IsUniqueViolation(errors.New("boom")) {
		t.Fatal("isUniqueViolation must reject non-pg errors")
	}
}

func TestAdminRepoUpdateDeleteRole(t *testing.T) {
	ctx := context.Background()
	// UpdateRole：成功（改名 + 描述 + ClearDescription）
	stub := &adminFullStub{
		queryRow: map[string][]any{
			queryRoleByID:   {"role_1", "旧名", "旧描述", aTestTime, aTestTime, false},
			queryRoleByName: {"role_1", "旧名", "旧描述", aTestTime, aTestTime, false},
		},
	}
	repo := newAdminFullRepo(stub)
	name := "新名"
	desc := "新描述"
	if err := repo.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_1", Name: &name, Description: &desc}); err != nil {
		t.Fatalf("update role: %v", err)
	}
	// ClearDescription 路径
	stub = &adminFullStub{queryRow: map[string][]any{queryRoleByID: {"role_1", "旧名", "旧描述", aTestTime, aTestTime, false}}}
	if err := newAdminFullRepo(stub).UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_1", ClearDescription: true}); err != nil {
		t.Fatalf("update role clear desc: %v", err)
	}
	// 角色不存在
	stub = &adminFullStub{queryRowErr: map[string]error{queryRoleByID: pgx.ErrNoRows}}
	if err := newAdminFullRepo(stub).UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "missing"}); err == nil {
		t.Fatal("update role missing must fail")
	}
	// 重名冲突（另一角色占名）
	stub = &adminFullStub{
		queryRow: map[string][]any{
			queryRoleByID:   {"role_1", "旧名", "", aTestTime, aTestTime, false},
			queryRoleByName: {"role_2", "新名", "", aTestTime, aTestTime, false},
		},
	}
	if err := newAdminFullRepo(stub).UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_1", Name: &name}); err == nil {
		t.Fatal("update role name conflict must fail")
	}
	// unique violation → 10400
	stub = &adminFullStub{
		queryRow: map[string][]any{queryRoleByID: {"role_1", "旧名", "", aTestTime, aTestTime, false}},
		execErrs: map[string]error{queryRoleUpdate: &pgconn.PgError{Code: "23505"}},
	}
	if err := newAdminFullRepo(stub).UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_1"}); err == nil {
		t.Fatal("update role unique violation must fail")
	}
	// getRoleByID 错误传播
	stub = &adminFullStub{queryRowErr: map[string]error{queryRoleByID: errors.New("boom")}}
	if err := newAdminFullRepo(stub).UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: "role_1"}); err == nil {
		t.Fatal("update role query error must propagate")
	}

	// DeleteRole：tx 成功
	stub = &adminFullStub{queryRow: map[string][]any{queryRoleByID: {"role_1", "旧名", "", aTestTime, aTestTime, false}}}
	if err := newAdminFullRepo(stub).DeleteRole(ctx, "role_1"); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	// 未找到
	stub = &adminFullStub{queryRowErr: map[string]error{queryRoleByID: pgx.ErrNoRows}}
	if err := newAdminFullRepo(stub).DeleteRole(ctx, "missing"); err == nil {
		t.Fatal("delete role missing must fail")
	}
	// 已软删 → 幂等成功
	stub = &adminFullStub{queryRow: map[string][]any{queryRoleByID: {"role_1", "旧名", "", aTestTime, aTestTime, true}}}
	if err := newAdminFullRepo(stub).DeleteRole(ctx, "role_1"); err != nil {
		t.Fatalf("delete role already deleted: %v", err)
	}
	// tx 内 exec 错误
	stub = &adminFullStub{
		queryRow: map[string][]any{queryRoleByID: {"role_1", "旧名", "", aTestTime, aTestTime, false}},
		execErrs: map[string]error{queryRoleDeleteAssignments: errors.New("boom")},
	}
	if err := newAdminFullRepo(stub).DeleteRole(ctx, "role_1"); err == nil {
		t.Fatal("delete role exec error must propagate")
	}
}

func TestAdminRepoAuditLogs(t *testing.T) {
	ctx := context.Background()
	entry := AuditLogEntry{
		LogID:      "audit_1",
		Action:     AuditActionRoleCreate,
		Operator:   UserInfo{UserID: "op1", Nickname: "操作者"},
		TargetType: AuditTargetTypeRole,
		TargetID:   "role_1",
		TargetName: "角色",
		Details:    AuditDetails{"k": "v"},
		CreatedAt:  aTestTime,
	}
	// AppendAuditLog：带 details
	if err := newAdminFullRepo(&adminFullStub{}).AppendAuditLog(ctx, entry); err != nil {
		t.Fatalf("append audit: %v", err)
	}
	// 无 details + 自动 ID/时间
	if err := newAdminFullRepo(&adminFullStub{}).AppendAuditLog(ctx, AuditLogEntry{Action: AuditActionRoleDelete}); err != nil {
		t.Fatalf("append audit minimal: %v", err)
	}
	// exec 错误
	stub := &adminFullStub{execErrs: map[string]error{queryAuditInsert: errors.New("boom")}}
	if err := newAdminFullRepo(stub).AppendAuditLog(ctx, entry); err == nil {
		t.Fatal("append audit exec error must propagate")
	}

	// ListAuditLogs：无 cursor 分页（3 行 limit=2）
	stub = &adminFullStub{queryRows: map[string][][]any{queryAuditList: {
		adminAuditRowVals("audit_3"), adminAuditRowVals("audit_2"), adminAuditRowVals("audit_1"),
	}}}
	page, err := newAdminFullRepo(stub).ListAuditLogs(ctx, AuditLogFilter{Pagination: types.PaginationParams{Limit: 2}}, true)
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("list audit: %#v err=%v", page, err)
	}
	if page.Items[0].Details == nil || page.Items[0].IPAddress == nil {
		t.Fatalf("audit item fields: %#v", page.Items[0])
	}
	// filter 全部命中 + cursor 路径（keyset：只保留 LogID < cursor 的条目）
	cursorToken := encodeAuditCursor(auditCursor{CreatedAt: aTestTime, LogID: "audit_2"})
	start := aTestTime.Add(-time.Hour)
	end := aTestTime.Add(time.Hour)
	stub = &adminFullStub{queryRows: map[string][][]any{queryAuditList: {adminAuditRowVals("audit_3"), adminAuditRowVals("audit_1")}}}
	page, err = newAdminFullRepo(stub).ListAuditLogs(ctx, AuditLogFilter{
		Action: AuditActionRoleCreate, OperatorID: "op1", TargetType: AuditTargetTypeRole, TargetID: "role_1",
		StartTime: &start, EndTime: &end,
		Pagination: types.PaginationParams{Limit: 10, Cursor: cursorToken},
	}, false)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list audit filtered: %#v err=%v", page, err)
	}
	// 无效 cursor
	if _, err := newAdminFullRepo(&adminFullStub{}).ListAuditLogs(ctx, AuditLogFilter{Pagination: types.PaginationParams{Cursor: "garbage"}}, true); err == nil {
		t.Fatal("list audit invalid cursor must fail")
	}
	// Query 错误
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAuditList: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).ListAuditLogs(ctx, AuditLogFilter{}, true); err == nil {
		t.Fatal("list audit query error must propagate")
	}
	// rows.Err
	stub = &adminFullStub{queryCustomRows: map[string]persistence.Rows{
		queryAuditList: &adminRowsErrValues{rows: [][]any{adminAuditRowVals("audit_1")}, err: errors.New("tail")},
	}}
	if _, err := newAdminFullRepo(stub).ListAuditLogs(ctx, AuditLogFilter{}, true); err == nil {
		t.Fatal("list audit tail error must propagate")
	}
}

type adminRowsErrValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *adminRowsErrValues) Next() bool { return r.idx < len(r.rows) }
func (r *adminRowsErrValues) Scan(dest ...any) error {
	row := adminRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *adminRowsErrValues) Close() error { return nil }
func (r *adminRowsErrValues) Err() error   { return r.err }

func TestAdminRepoUsers(t *testing.T) {
	ctx := context.Background()
	// SearchUsers：正常 + 分页 cursor
	userCursor := encodeAdminUserCursor(adminUserCursor{CreatedAt: aTestTime, UserID: "u1"})
	stub := &adminFullStub{queryRows: map[string][][]any{queryAdminSearchUsers: {adminUserRowVals("u3"), adminUserRowVals("u2"), adminUserRowVals("u1")}}}
	page, err := newAdminFullRepo(stub).SearchUsers(ctx, AdminUserSearchRequest{
		UserID: "u", Pagination: &types.PaginationParams{Limit: 2, Cursor: userCursor},
	})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("search users: %#v err=%v", page, err)
	}
	if page.Items[0].Location == nil || page.Items[0].Email == nil || page.Items[0].CreatedAt == "" {
		t.Fatalf("user item fields: %#v", page.Items[0])
	}
	// 无 cursor + 分页触发 Next
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchUsers: {adminUserRowVals("u3"), adminUserRowVals("u2"), adminUserRowVals("u1")}}}
	page, err = newAdminFullRepo(stub).SearchUsers(ctx, AdminUserSearchRequest{Pagination: &types.PaginationParams{Limit: 2}})
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("search users next: %#v err=%v", page, err)
	}
	// 无效 cursor
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchUsers: {adminUserRowVals("u1")}}}
	if _, err := newAdminFullRepo(stub).SearchUsers(ctx, AdminUserSearchRequest{Pagination: &types.PaginationParams{Cursor: "garbage"}}); err == nil {
		t.Fatal("search users invalid cursor must fail")
	}
	// Query 错误
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAdminSearchUsers: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).SearchUsers(ctx, AdminUserSearchRequest{}); err == nil {
		t.Fatal("search users query error must propagate")
	}
	// rows.Err
	stub = &adminFullStub{queryCustomRows: map[string]persistence.Rows{
		queryAdminSearchUsers: &adminRowsErrValues{rows: [][]any{adminUserRowVals("u1")}, err: errors.New("tail")},
	}}
	if _, err := newAdminFullRepo(stub).SearchUsers(ctx, AdminUserSearchRequest{}); err == nil {
		t.Fatal("search users tail error must propagate")
	}

	// GetUser：命中 / 未命中 / 错误
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetUser: adminUserRowVals("u1")}}
	if item, found, err := newAdminFullRepo(stub).GetUser(ctx, "u1"); err != nil || !found || item.User.UserID != "u1" {
		t.Fatalf("get user: %#v found=%v err=%v", item, found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetUser: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).GetUser(ctx, "missing"); err != nil || found {
		t.Fatalf("get user missing: found=%v err=%v", found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetUser: errors.New("boom")}}
	if _, _, err := newAdminFullRepo(stub).GetUser(ctx, "u1"); err == nil {
		t.Fatal("get user error must propagate")
	}

	// GetUserBanState：命中带 bannedAt / 未命中 / 错误
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetUserBanState: {string(UserAccountStatusBanned), "理由", 30, aTestTime}}}
	state, found, err := newAdminFullRepo(stub).GetUserBanState(ctx, "u1")
	if err != nil || !found || state.Status != UserAccountStatusBanned || state.BannedAt.IsZero() {
		t.Fatalf("get ban state: %#v found=%v err=%v", state, found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetUserBanState: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).GetUserBanState(ctx, "u1"); err != nil || found {
		t.Fatalf("get ban state missing: found=%v err=%v", found, err)
	}

	// SetUserBanState：banned 路径 + normal 路径
	stub = &adminFullStub{}
	if err := newAdminFullRepo(stub).SetUserBanState(ctx, "u1", UserBanState{Status: UserAccountStatusBanned, Reason: "理由", DurationDays: 7}); err != nil {
		t.Fatalf("set ban: %v", err)
	}
	if err := newAdminFullRepo(stub).SetUserBanState(ctx, "u1", UserBanState{Status: UserAccountStatusNormal}); err != nil {
		t.Fatalf("set normal: %v", err)
	}
	// banned 且 BannedAt 已给
	if err := newAdminFullRepo(stub).SetUserBanState(ctx, "u1", UserBanState{Status: UserAccountStatusBanned, BannedAt: aTestTime}); err != nil {
		t.Fatalf("set ban with time: %v", err)
	}
	// 空状态 → normal
	if err := newAdminFullRepo(stub).SetUserBanState(ctx, "u1", UserBanState{}); err != nil {
		t.Fatalf("set empty state: %v", err)
	}
	stub = &adminFullStub{execErrs: map[string]error{queryAdminSetUserBanState: errors.New("boom")}}
	if err := newAdminFullRepo(stub).SetUserBanState(ctx, "u1", UserBanState{Status: UserAccountStatusBanned}); err == nil {
		t.Fatal("set ban exec error must propagate")
	}

	// ListUserSessions：2 行 / Query 错误
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminListUserSessions: {{"s1"}, {"s2"}}}}
	sessions, err := newAdminFullRepo(stub).ListUserSessions(ctx, "u1")
	if err != nil || len(sessions) != 2 {
		t.Fatalf("list sessions: %#v err=%v", sessions, err)
	}
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAdminListUserSessions: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).ListUserSessions(ctx, "u1"); err == nil {
		t.Fatal("list sessions query error must propagate")
	}

	// RevokeSession
	stub = &adminFullStub{}
	if err := newAdminFullRepo(stub).RevokeSession(ctx, "s1"); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	stub = &adminFullStub{execErrs: map[string]error{queryAdminRevokeSession: errors.New("boom")}}
	if err := newAdminFullRepo(stub).RevokeSession(ctx, "s1"); err == nil {
		t.Fatal("revoke session exec error must propagate")
	}

	// IsBanned：banned / normal / 未找到
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetUserBanState: {string(UserAccountStatusBanned), "", 0, aTestTime}}}
	if banned, err := newAdminFullRepo(stub).IsBanned(ctx, "u1"); err != nil || !banned {
		t.Fatalf("is banned: %v err=%v", banned, err)
	}
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetUserBanState: {string(UserAccountStatusNormal), "", 0, aTestTime}}}
	if banned, err := newAdminFullRepo(stub).IsBanned(ctx, "u1"); err != nil || banned {
		t.Fatalf("is banned normal: %v err=%v", banned, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetUserBanState: pgx.ErrNoRows}}
	if banned, err := newAdminFullRepo(stub).IsBanned(ctx, "u1"); err != nil || banned {
		t.Fatalf("is banned missing: %v err=%v", banned, err)
	}
}

func TestAdminRepoGroups(t *testing.T) {
	ctx := context.Background()
	// SearchGroups：正常 + cursor 分页
	groupCursor := encodeAdminGroupCursor(adminGroupCursor{CreatedAt: aTestTime, GroupID: "g1"})
	stub := &adminFullStub{queryRows: map[string][][]any{queryAdminSearchGroups: {adminGroupRowVals17("g3"), adminGroupRowVals17("g2"), adminGroupRowVals17("g1")}}}
	page, err := newAdminFullRepo(stub).SearchGroups(ctx, AdminGroupSearchRequest{
		Name: "群", Pagination: &types.PaginationParams{Limit: 2, Cursor: groupCursor},
	})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("search groups: %#v err=%v", page, err)
	}
	if page.Items[0].Owner.Nickname == "" || page.Items[0].CreatedAt == "" {
		t.Fatalf("group item fields: %#v", page.Items[0])
	}
	// 无 cursor 触发 Next
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchGroups: {adminGroupRowVals17("g3"), adminGroupRowVals17("g2"), adminGroupRowVals17("g1")}}}
	page, err = newAdminFullRepo(stub).SearchGroups(ctx, AdminGroupSearchRequest{Pagination: &types.PaginationParams{Limit: 2}})
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("search groups next: %#v err=%v", page, err)
	}
	// 无效 cursor
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchGroups: {adminGroupRowVals17("g1")}}}
	if _, err := newAdminFullRepo(stub).SearchGroups(ctx, AdminGroupSearchRequest{Pagination: &types.PaginationParams{Cursor: "garbage"}}); err == nil {
		t.Fatal("search groups invalid cursor must fail")
	}
	// Query 错误
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAdminSearchGroups: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).SearchGroups(ctx, AdminGroupSearchRequest{}); err == nil {
		t.Fatal("search groups query error must propagate")
	}

	// GetGroup：命中 / 未命中
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetGroup: adminGroupRowVals17("g1")}}
	if item, found, err := newAdminFullRepo(stub).GetGroup(ctx, "g1"); err != nil || !found || item.Group.GroupID != "g1" {
		t.Fatalf("get group: %#v found=%v err=%v", item, found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetGroup: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).GetGroup(ctx, "missing"); err != nil || found {
		t.Fatalf("get group missing: found=%v err=%v", found, err)
	}

	// DismissGroup：成功 / 未找到 / exec 错误
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetGroup: adminGroupRowVals17("g1")}}
	item, found, err := newAdminFullRepo(stub).DismissGroup(ctx, "g1")
	if err != nil || !found || item.Group.GroupID != "g1" {
		t.Fatalf("dismiss group: %#v found=%v err=%v", item, found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetGroup: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).DismissGroup(ctx, "missing"); err != nil || found {
		t.Fatalf("dismiss group missing: found=%v err=%v", found, err)
	}
	stub = &adminFullStub{
		queryRow: map[string][]any{queryAdminGetGroup: adminGroupRowVals17("g1")},
		execErrs: map[string]error{queryAdminDismissGroup: errors.New("boom")},
	}
	if _, _, err := newAdminFullRepo(stub).DismissGroup(ctx, "g1"); err == nil {
		t.Fatal("dismiss group exec error must propagate")
	}

	// ListGroupMemberIDs
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminListGroupMemberIDs: {{"u1"}, {"u2"}}}}
	ids, err := newAdminFullRepo(stub).ListGroupMemberIDs(ctx, "g1")
	if err != nil || len(ids) != 2 {
		t.Fatalf("list member ids: %#v err=%v", ids, err)
	}
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAdminListGroupMemberIDs: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).ListGroupMemberIDs(ctx, "g1"); err == nil {
		t.Fatal("list member ids query error must propagate")
	}
}

func TestAdminRepoReports(t *testing.T) {
	ctx := context.Background()
	// SearchReports：正常 + 分页（keyset：只保留 ReportID < cursor 的条目）
	reportCursor := encodeAdminReportCursor(adminReportCursor{CreatedAt: aTestTime, ReportID: "r3"})
	stub := &adminFullStub{queryRows: map[string][][]any{queryAdminSearchReports: {adminReportRowVals16("r3"), adminReportRowVals16("r2"), adminReportRowVals16("r1")}}}
	page, err := newAdminFullRepo(stub).SearchReports(ctx, AdminReportListRequest{
		Status: ReportStatusPending, Pagination: &types.PaginationParams{Limit: 2, Cursor: reportCursor},
	})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("search reports: %#v err=%v", page, err)
	}
	// 无 cursor → Next
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchReports: {adminReportRowVals16("r3"), adminReportRowVals16("r2"), adminReportRowVals16("r1")}}}
	page, err = newAdminFullRepo(stub).SearchReports(ctx, AdminReportListRequest{Pagination: &types.PaginationParams{Limit: 2}})
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("search reports next: %#v err=%v", page, err)
	}
	// 无效 cursor
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchReports: {adminReportRowVals16("r1")}}}
	if _, err := newAdminFullRepo(stub).SearchReports(ctx, AdminReportListRequest{Pagination: &types.PaginationParams{Cursor: "garbage"}}); err == nil {
		t.Fatal("search reports invalid cursor must fail")
	}
	// Query 错误
	stub = &adminFullStub{queryRowsErr: map[string]error{queryAdminSearchReports: errors.New("boom")}}
	if _, err := newAdminFullRepo(stub).SearchReports(ctx, AdminReportListRequest{}); err == nil {
		t.Fatal("search reports query error must propagate")
	}
	// 无效 snapshot json
	stub = &adminFullStub{queryRows: map[string][][]any{queryAdminSearchReports: {{"r1", "moment", "m1", "理由", "描述", "pending", aTestTime, nil, "u9", "举报人", "", "", "", "", "", "not-json"}}}}
	if _, err := newAdminFullRepo(stub).SearchReports(ctx, AdminReportListRequest{}); err == nil {
		t.Fatal("search reports bad snapshot must fail")
	}

	// GetReportHandleRecord：命中（withTargetUser 19 列）/ 未命中
	reportRow19 := append(append([]any{}, adminReportRowVals16("r1")...), "t1", "目标", "")
	stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetReportHandleRecord: reportRow19}}
	record, found, err := newAdminFullRepo(stub).GetReportHandleRecord(ctx, "r1")
	if err != nil || !found || record.TargetUser.UserID != "t1" {
		t.Fatalf("get report record: %#v found=%v err=%v", record, found, err)
	}
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetReportHandleRecord: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).GetReportHandleRecord(ctx, "missing"); err != nil || found {
		t.Fatalf("get report record missing: found=%v err=%v", found, err)
	}

	// HandleReport：resolve / dismiss / 带 handledBy / exec 错误
	handledBy := &UserInfo{UserID: "admin1", Nickname: "管理员"}
	for _, action := range []ReportAction{ReportActionResolve, ReportActionDismiss} {
		stub = &adminFullStub{queryRow: map[string][]any{queryAdminGetReportHandleRecord: reportRow19}}
		item, found, err := newAdminFullRepo(stub).HandleReport(ctx, "r1", action, "处理完毕", handledBy)
		if err != nil || !found || item.HandledAt == nil || item.HandledBy == nil || item.HandleNote == nil {
			t.Fatalf("handle report %s: %#v found=%v err=%v", action, item, found, err)
		}
	}
	// 未找到
	stub = &adminFullStub{queryRowErr: map[string]error{queryAdminGetReportHandleRecord: pgx.ErrNoRows}}
	if _, found, err := newAdminFullRepo(stub).HandleReport(ctx, "r1", ReportActionResolve, "", nil); err != nil || found {
		t.Fatalf("handle report missing: found=%v err=%v", found, err)
	}
	// exec 错误
	stub = &adminFullStub{
		queryRow: map[string][]any{queryAdminGetReportHandleRecord: reportRow19},
		execErrs: map[string]error{queryAdminHandleReport: errors.New("boom")},
	}
	if _, _, err := newAdminFullRepo(stub).HandleReport(ctx, "r1", ReportActionResolve, "", nil); err == nil {
		t.Fatal("handle report exec error must propagate")
	}
}

func TestAdminRepoRecallDeleteContent(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(repo *PostgresRepository) error
		exec string
	}{
		{"recall group", func(r *PostgresRepository) error { return r.RecallGroupMessage(ctx, "m1") }, "UPDATE group_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE"},
		{"recall private", func(r *PostgresRepository) error { return r.RecallPrivateMessage(ctx, "m1") }, "UPDATE private_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE"},
		{"delete moment", func(r *PostgresRepository) error { return r.DeleteMoment(ctx, "m1") }, "DELETE FROM moments WHERE moment_id = $1"},
		{"delete comment", func(r *PostgresRepository) error { return r.DeleteComment(ctx, "c1") }, "DELETE FROM comments WHERE comment_id = $1"},
	}
	for _, tc := range cases {
		if err := tc.call(newAdminFullRepo(&adminFullStub{})); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		stub := &adminFullStub{execErrs: map[string]error{tc.exec: errors.New("boom")}}
		if err := tc.call(newAdminFullRepo(stub)); err == nil {
			t.Fatalf("%s exec error must propagate", tc.name)
		}
	}
}

func TestAdminModerationStorePersistence(t *testing.T) {
	ctx := context.Background()
	// SearchTags：分页 + cursor
	store := newAdminFullStore(&adminFullStub{queryRows: map[string][][]any{querySearchTags: {
		{"tag3", 3, aTestTime}, {"tag2", 2, aTestTime}, {"tag1", 1, aTestTime},
	}}})
	page, err := store.SearchTags(ctx, TagSearchFilter{Keyword: "tag", Limit: 2, Cursor: &tagCursor{Name: "tag1", UsageCount: 1, CreatedAt: aTestTime}})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("search tags: %#v err=%v", page, err)
	}
	// 无 cursor → Next
	store = newAdminFullStore(&adminFullStub{queryRows: map[string][][]any{querySearchTags: {
		{"tag3", 3, aTestTime}, {"tag2", 2, aTestTime}, {"tag1", 1, aTestTime},
	}}})
	page, err = store.SearchTags(ctx, TagSearchFilter{Keyword: " tag ", Limit: 2})
	if err != nil || len(page.Items) != 2 || page.Next == nil {
		t.Fatalf("search tags next: %#v err=%v", page, err)
	}
	// Query 错误
	store = newAdminFullStore(&adminFullStub{queryRowsErr: map[string]error{querySearchTags: errors.New("boom")}})
	if _, err := store.SearchTags(ctx, TagSearchFilter{}); err == nil {
		t.Fatal("search tags query error must propagate")
	}
	// rows.Err
	store = newAdminFullStore(&adminFullStub{queryCustomRows: map[string]persistence.Rows{
		querySearchTags: &adminRowsErrValues{rows: [][]any{{"tag1", 1, aTestTime}}, err: errors.New("tail")},
	}})
	if _, err := store.SearchTags(ctx, TagSearchFilter{}); err == nil {
		t.Fatal("search tags tail error must propagate")
	}
	// 回退内存路径（queryable 不可用，内存 store 含预置标签）
	store = NewPersistenceModerationStore(&persistence.Factory{}, nil)
	if page, err := store.SearchTags(ctx, TagSearchFilter{Limit: 5}); err != nil {
		t.Fatalf("search tags fallback: %#v err=%v", page, err)
	}

	// DeleteTag 分支：无引用 → not found；已屏蔽 → 幂等；硬删 + 屏蔽
	store = newAdminFullStore(&adminFullStub{queryRow: map[string][]any{queryTagDeleteState: {false, false}}})
	if err := store.DeleteTag(ctx, "tag1", false); !errors.Is(err, errTagNotFound) {
		t.Fatalf("delete tag not found: %v", err)
	}
	store = newAdminFullStore(&adminFullStub{queryRow: map[string][]any{queryTagDeleteState: {true, false}}})
	if err := store.DeleteTag(ctx, "tag1", false); err != nil {
		t.Fatalf("delete tag blocked: %v", err)
	}
	store = newAdminFullStore(&adminFullStub{queryRow: map[string][]any{queryTagDeleteState: {false, true}}})
	if err := store.DeleteTag(ctx, "tag1", true); err != nil {
		t.Fatalf("delete tag hard: %v", err)
	}
	store = newAdminFullStore(&adminFullStub{queryRow: map[string][]any{queryTagDeleteState: {false, true}}})
	if err := store.DeleteTag(ctx, "tag1", false); err != nil {
		t.Fatalf("delete tag soft: %v", err)
	}
	// GetTagDeleteState 查询错误
	store = newAdminFullStore(&adminFullStub{queryRowErr: map[string]error{queryTagDeleteState: errors.New("boom")}})
	if err := store.DeleteTag(ctx, "tag1", false); err == nil {
		t.Fatal("delete tag query error must propagate")
	}

	// 公告方法走 system adapter（NewPersistenceModerationStore 第二个参数 nil 时自动创建）
	store = NewPersistenceModerationStore(store.factory, nil)
	ann, err := store.CreateAnnouncement(ctx, systemmodule.AnnouncementCreateInput{Title: "公告", Content: "内容"})
	if err != nil || ann.AnnouncementID == "" {
		t.Fatalf("create announcement: %#v err=%v", ann, err)
	}
	if _, _, err := store.GetAnnouncement(ctx, ann.AnnouncementID); err != nil {
		t.Fatalf("get announcement: %v", err)
	}
	title := "新标题"
	if _, err := store.UpdateAnnouncement(ctx, systemmodule.AnnouncementUpdateInput{AnnouncementID: ann.AnnouncementID, Title: &title}); err != nil {
		t.Fatalf("update announcement: %v", err)
	}
	if err := store.DeleteAnnouncement(ctx, ann.AnnouncementID); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	// adapter 的独立方法（GetMeta/Health/ListAnnouncements；内存无 seed 公告，
	// ListAnnouncements 返回空列表但不报错）
	adapter := NewInMemoryModerationStoreSystemAdapter()
	meta := adapter.GetMeta(ctx)
	if meta.Version != "v1" {
		t.Fatalf("get meta: %#v", meta)
	}
	_, _, status := adapter.Health(ctx)
	if status == nil || status.Status == "" {
		t.Fatal("health must return status")
	}
	if _, err := adapter.ListAnnouncements(ctx, 5); err != nil {
		t.Fatalf("list announcements: %v", err)
	}
	if normalizeTagSearchKeyword("  kw  ") != "kw" {
		t.Fatal("normalizeTagSearchKeyword")
	}
}

func TestAdminStoreCursorHelpers(t *testing.T) {
	now := aTestTime
	// encode/decode round trip
	for _, tc := range []struct {
		encode func() string
		decode func(string) error
	}{
		{func() string { return encodeAdminCursor(adminCursor{GrantedAt: now, UserID: "u1"}) }, func(s string) error {
			_, err := decodeAdminCursor(s)
			if err != nil {
				return err
			}
			return nil
		}},
		{func() string { return encodeAuditCursor(auditCursor{CreatedAt: now, LogID: "l1"}) }, func(s string) error {
			_, err := decodeAuditCursor(s)
			if err != nil {
				return err
			}
			return nil
		}},
		{func() string { return encodeAdminUserCursor(adminUserCursor{CreatedAt: now, UserID: "u1"}) }, func(s string) error {
			_, err := decodeAdminUserCursor(s)
			if err != nil {
				return err
			}
			return nil
		}},
		{func() string { return encodeAdminGroupCursor(adminGroupCursor{CreatedAt: now, GroupID: "g1"}) }, func(s string) error {
			_, err := decodeAdminGroupCursor(s)
			if err != nil {
				return err
			}
			return nil
		}},
		{func() string { return encodeAdminReportCursor(adminReportCursor{CreatedAt: now, ReportID: "r1"}) }, func(s string) error {
			_, err := decodeAdminReportCursor(s)
			if err != nil {
				return err
			}
			return nil
		}},
	} {
		token := tc.encode()
		if token == "" || tc.decode(token) != nil {
			t.Fatalf("cursor round trip failed: %q", token)
		}
		// 空串 → 无错误
		if err := tc.decode(""); err != nil {
			t.Fatalf("empty cursor must decode: %v", err)
		}
		// 非法 base64
		if err := tc.decode("!!!not-base64!!!"); err == nil {
			t.Fatal("bad base64 must fail")
		}
		// 缺字段（编码只含部分字段）
		blob, _ := json.Marshal(map[string]string{})
		badToken := base64.RawURLEncoding.EncodeToString(blob)
		if err := tc.decode(badToken); err == nil {
			t.Fatal("incomplete cursor must fail")
		}
	}
	// normalizeLimit
	if types.NormalizeLimit(0) != 20 || types.NormalizeLimit(101) != 100 || types.NormalizeLimit(5) != 5 {
		t.Fatal("normalizeLimit")
	}
	// limitPageItems
	items := []int{1, 2, 3}
	if got, truncated := limitPageItems(items, 2); len(got) != 2 || !truncated {
		t.Fatal("limitPageItems truncated")
	}
	if got, truncated := limitPageItems(items, 5); len(got) != 3 || truncated {
		t.Fatal("limitPageItems full")
	}
}

// 避免 import 未使用：strings 由 cursor helper 校验外的工具路径引用。
var _ = strings.TrimSpace
