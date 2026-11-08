package admin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	systemmodule "hotspot/backend-go/internal/modules/system"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/types"
)

// 本文件补齐 admin Service 层在「权限/输入校验通过后，仓储或跨模块依赖失败」
// 时的错误分支，以及分页游标、PII 脱敏、举报裁决编排的可达边界。
// 全部使用内存仓储与可控替身，无外部依赖。

// covRepo 在内存仓储之上按方法注入错误/自定义分页结果。
type covRepo struct {
	*InMemoryRepository
	getActorErrUser       string
	getActorErr           error
	listPermissionsErr    error
	listRolesErr          error
	getRoleErr            error
	createRoleErr         error
	updateRoleErr         error
	deleteRoleErr         error
	setRolePermissionsErr error
	listAdminsErr         error
	listAdminsPage        *adminListPage
	grantAdminErr         error
	revokeAdminErr        error
	listAuditLogsErr      error
	searchUsersErr        error
	searchUsersPage       *adminUserPage
	getUserErr            error
	userBanStateErr       error
	searchGroupsErr       error
	searchGroupsPage      *adminGroupPage
	getGroupErr           error
	searchReportsErr      error
	searchReportsPage     *adminReportPage
	getReportRecordErr    error
	getReportRecord       *reportHandleRecord
	auditErr              error
	statsGeoErr           error
	statsDrilldownErr     error
}

func (r *covRepo) GetActor(ctx context.Context, userID string) (AdminActor, bool, error) {
	if r.getActorErr != nil && (r.getActorErrUser == "" || r.getActorErrUser == userID) {
		return AdminActor{}, false, r.getActorErr
	}
	return r.InMemoryRepository.GetActor(ctx, userID)
}

func (r *covRepo) ListPermissions(ctx context.Context) ([]AdminPermissionItem, error) {
	if r.listPermissionsErr != nil {
		return nil, r.listPermissionsErr
	}
	return r.InMemoryRepository.ListPermissions(ctx)
}

func (r *covRepo) ListRoles(ctx context.Context) ([]AdminRoleItem, error) {
	if r.listRolesErr != nil {
		return nil, r.listRolesErr
	}
	return r.InMemoryRepository.ListRoles(ctx)
}

func (r *covRepo) GetRole(ctx context.Context, roleID string) (AdminRoleItem, bool, error) {
	if r.getRoleErr != nil {
		return AdminRoleItem{}, false, r.getRoleErr
	}
	return r.InMemoryRepository.GetRole(ctx, roleID)
}

func (r *covRepo) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	if r.createRoleErr != nil {
		return AdminRoleItem{}, r.createRoleErr
	}
	return r.InMemoryRepository.CreateRole(ctx, req)
}

func (r *covRepo) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) error {
	if r.updateRoleErr != nil {
		return r.updateRoleErr
	}
	return r.InMemoryRepository.UpdateRole(ctx, req)
}

func (r *covRepo) DeleteRole(ctx context.Context, roleID string) error {
	if r.deleteRoleErr != nil {
		return r.deleteRoleErr
	}
	return r.InMemoryRepository.DeleteRole(ctx, roleID)
}

func (r *covRepo) SetRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	if r.setRolePermissionsErr != nil {
		return r.setRolePermissionsErr
	}
	return r.InMemoryRepository.SetRolePermissions(ctx, roleID, permissionKeys)
}

func (r *covRepo) ListAdmins(ctx context.Context, pagination PaginationParams) (adminListPage, error) {
	if r.listAdminsErr != nil {
		return adminListPage{}, r.listAdminsErr
	}
	if r.listAdminsPage != nil {
		return *r.listAdminsPage, nil
	}
	return r.InMemoryRepository.ListAdmins(ctx, pagination)
}

func (r *covRepo) GrantAdmin(ctx context.Context, grant AdminGrantUpsert) error {
	if r.grantAdminErr != nil {
		return r.grantAdminErr
	}
	return r.InMemoryRepository.GrantAdmin(ctx, grant)
}

func (r *covRepo) RevokeAdmin(ctx context.Context, userID string) error {
	if r.revokeAdminErr != nil {
		return r.revokeAdminErr
	}
	return r.InMemoryRepository.RevokeAdmin(ctx, userID)
}

func (r *covRepo) ListAuditLogs(ctx context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	if r.listAuditLogsErr != nil {
		return auditLogPage{}, r.listAuditLogsErr
	}
	return r.InMemoryRepository.ListAuditLogs(ctx, filter, includeIPAddress)
}

func (r *covRepo) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	if r.searchUsersErr != nil {
		return adminUserPage{}, r.searchUsersErr
	}
	if r.searchUsersPage != nil {
		return *r.searchUsersPage, nil
	}
	return r.InMemoryRepository.SearchUsers(ctx, req)
}

func (r *covRepo) GetUser(ctx context.Context, userID string) (AdminUserItem, bool, error) {
	if r.getUserErr != nil {
		return AdminUserItem{}, false, r.getUserErr
	}
	return r.InMemoryRepository.GetUser(ctx, userID)
}

func (r *covRepo) GetUserBanState(ctx context.Context, userID string) (UserBanState, bool, error) {
	if r.userBanStateErr != nil {
		return UserBanState{}, false, r.userBanStateErr
	}
	return r.InMemoryRepository.GetUserBanState(ctx, userID)
}

func (r *covRepo) SearchGroups(ctx context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	if r.searchGroupsErr != nil {
		return adminGroupPage{}, r.searchGroupsErr
	}
	if r.searchGroupsPage != nil {
		return *r.searchGroupsPage, nil
	}
	return r.InMemoryRepository.SearchGroups(ctx, req)
}

func (r *covRepo) GetGroup(ctx context.Context, groupID string) (AdminGroupItem, bool, error) {
	if r.getGroupErr != nil {
		return AdminGroupItem{}, false, r.getGroupErr
	}
	return r.InMemoryRepository.GetGroup(ctx, groupID)
}

func (r *covRepo) SearchReports(ctx context.Context, req AdminReportListRequest) (adminReportPage, error) {
	if r.searchReportsErr != nil {
		return adminReportPage{}, r.searchReportsErr
	}
	if r.searchReportsPage != nil {
		return *r.searchReportsPage, nil
	}
	return r.InMemoryRepository.SearchReports(ctx, req)
}

func (r *covRepo) GetReportHandleRecord(ctx context.Context, reportID string) (reportHandleRecord, bool, error) {
	if r.getReportRecordErr != nil {
		return reportHandleRecord{}, false, r.getReportRecordErr
	}
	if r.getReportRecord != nil {
		return *r.getReportRecord, true, nil
	}
	return r.InMemoryRepository.GetReportHandleRecord(ctx, reportID)
}

func (r *covRepo) AppendAuditLog(ctx context.Context, entry AuditLogEntry) error {
	if r.auditErr != nil {
		return r.auditErr
	}
	return r.InMemoryRepository.AppendAuditLog(ctx, entry)
}

func (r *covRepo) StatsGeo(ctx context.Context, query geoStatsQuery) (AdminGeoStatsResponse, error) {
	if r.statsGeoErr != nil {
		return AdminGeoStatsResponse{}, r.statsGeoErr
	}
	return r.InMemoryRepository.StatsGeo(ctx, query)
}

func (r *covRepo) StatsGeoDrilldown(ctx context.Context, query geoDrilldownQuery) (geoDrilldownPage, error) {
	if r.statsDrilldownErr != nil {
		return geoDrilldownPage{}, r.statsDrilldownErr
	}
	return r.InMemoryRepository.StatsGeoDrilldown(ctx, query)
}

// covModStore 在内存审核存储之上注入错误与自定义分页。
type covModStore struct {
	*InMemoryModerationStore
	searchTagsErr  error
	searchTagsPage *TagPage
	setTagBanErr   error
	searchAnnErr   error
	searchAnnPage  *systemmodule.AnnouncementPage
	getAnnErr      error
	createAnnErr   error
	updateAnnErr   error
	deleteAnnErr   error
}

func (s *covModStore) SearchTags(ctx context.Context, filter TagSearchFilter) (TagPage, error) {
	if s.searchTagsErr != nil {
		return TagPage{}, s.searchTagsErr
	}
	if s.searchTagsPage != nil {
		return *s.searchTagsPage, nil
	}
	return s.InMemoryModerationStore.SearchTags(ctx, filter)
}

func (s *covModStore) SetTagBan(ctx context.Context, name string, active, hardDelete bool) error {
	if s.setTagBanErr != nil {
		return s.setTagBanErr
	}
	return s.InMemoryModerationStore.SetTagBan(ctx, name, active, hardDelete)
}

func (s *covModStore) SearchAnnouncements(ctx context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	if s.searchAnnErr != nil {
		return systemmodule.AnnouncementPage{}, s.searchAnnErr
	}
	if s.searchAnnPage != nil {
		return *s.searchAnnPage, nil
	}
	return s.InMemoryModerationStore.SearchAnnouncements(ctx, filter)
}

func (s *covModStore) GetAnnouncement(ctx context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	if s.getAnnErr != nil {
		return systemmodule.AnnouncementRecord{}, false, s.getAnnErr
	}
	return s.InMemoryModerationStore.GetAnnouncement(ctx, announcementID)
}

func (s *covModStore) CreateAnnouncement(ctx context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	if s.createAnnErr != nil {
		return systemmodule.AnnouncementRecord{}, s.createAnnErr
	}
	return s.InMemoryModerationStore.CreateAnnouncement(ctx, input)
}

func (s *covModStore) UpdateAnnouncement(ctx context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	if s.updateAnnErr != nil {
		return systemmodule.AnnouncementRecord{}, s.updateAnnErr
	}
	return s.InMemoryModerationStore.UpdateAnnouncement(ctx, input)
}

func (s *covModStore) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	if s.deleteAnnErr != nil {
		return s.deleteAnnErr
	}
	return s.InMemoryModerationStore.DeleteAnnouncement(ctx, announcementID)
}

// covPush 覆盖公告推送与举报结果通知的错误出口。
type covPush struct {
	resolvedErr error
	dismissErr  error
}

func (p covPush) PushSystemAnnouncement(context.Context, systemmodule.SystemAnnouncementItem, string) {
}

func (p covPush) AppendReportResolved(context.Context, string, string, string, string, string) (string, error) {
	if p.resolvedErr != nil {
		return "", p.resolvedErr
	}
	return "cov_notify_resolved", nil
}

func (p covPush) AppendReportDismissed(context.Context, string, string, string, string, string) (string, error) {
	if p.dismissErr != nil {
		return "", p.dismissErr
	}
	return "cov_notify_dismissed", nil
}

// covOutbox 注入入队错误。
type covOutbox struct{ err error }

func (o covOutbox) Enqueue(context.Context, string, any) error { return o.err }

// covRunInTx 直接执行回调（模拟事务）。
func covRunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// covReportAdminStore 覆盖举报终态写入的错误出口。
type covReportAdminStore struct {
	updated bool
	err     error
}

func (s covReportAdminStore) MarkHandledAsAdmin(context.Context, string, string, time.Time, *string, *string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.updated, nil
}

func (s covReportAdminStore) DeleteHandledBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

// covSessionAdmin 覆盖会话撤销失败分支。
type covSessionAdmin struct {
	listErr   error
	revokeErr error
}

func (a covSessionAdmin) ListActiveSessionIDsAsAdmin(context.Context, string) ([]string, error) {
	if a.listErr != nil {
		return nil, a.listErr
	}
	return []string{"sess_1", "sess_2"}, nil
}

func (a covSessionAdmin) RevokeSessionAsAdmin(context.Context, string) (bool, error) {
	if a.revokeErr != nil {
		return false, a.revokeErr
	}
	return true, nil
}

// covBanWriter 覆盖账户封禁状态写入错误。
type covBanWriter struct{ err error }

func (w covBanWriter) SetAccountBanState(context.Context, string, usersmodule.AccountBanState) error {
	return w.err
}

// covMomentModerator / covCommentModerator / covMessageModerator 是错误可注入替身。
type covMomentModerator struct{ err error }

func (m covMomentModerator) DeleteAsAdmin(context.Context, string) error { return m.err }

type covCommentModerator struct{ err error }

func (m covCommentModerator) DeleteAsAdmin(context.Context, string) error { return m.err }

type covMessageModerator struct {
	hit bool
	err error
}

func (m covMessageModerator) RecallMessageByIDAsAdmin(context.Context, string, string) (bool, error) {
	return m.hit, m.err
}

type covGroupModerator struct {
	err error
}

func (m covGroupModerator) DismissAsAdmin(context.Context, string, string, string) (groupsmodule.GroupInfo, error) {
	return groupsmodule.GroupInfo{}, m.err
}

func covPermsService(repo Repository, mods ModerationStore, push announcementPushService) *Service {
	return NewService(repo, &stubProvider{config: defaultConfig()}, mods, push)
}

func covGrantPermissions(t *testing.T, repo *InMemoryRepository, userID, roleName string, permissions ...string) {
	t.Helper()
	ctx := context.Background()
	role, err := repo.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: roleName})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := repo.SetRolePermissions(ctx, role.RoleID, permissions); err != nil {
		t.Fatalf("set role permissions: %v", err)
	}
	if err := repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: userID, RoleIDs: []string{role.RoleID}}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────

func TestServiceRBACRepositoryErrors(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	repo.listPermissionsErr = errCovAdmin
	if _, apiErr, err := svc.ListPermissions(ctx); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("ListPermissions: apiErr=%v err=%v", apiErr, err)
	}
	repo.listPermissionsErr = nil
	repo.listRolesErr = errCovAdmin
	if _, apiErr, err := svc.ListRoles(ctx); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("ListRoles: apiErr=%v err=%v", apiErr, err)
	}
	repo.listRolesErr = nil

	// CreateRole 成功但审计写失败：角色仍创建成功（审计失败仅留痕）。
	repo.auditErr = errCovAdmin
	if _, apiErr, err := svc.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "cov_audit_role"}); apiErr != nil || err != nil {
		t.Fatalf("CreateRole with audit failure: apiErr=%v err=%v", apiErr, err)
	}
	repo.auditErr = nil

	// ListAdmins：仓储错误分别走 APIError 与原始错误两条出口。
	repo.listAdminsErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.ListAdmins(ctx, AdminRbacAdminsListRequest{}); apiErr == nil || err != nil {
		t.Fatalf("ListAdmins api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.listAdminsErr = errCovAdmin
	if _, apiErr, err := svc.ListAdmins(ctx, AdminRbacAdminsListRequest{}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("ListAdmins raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.listAdminsErr = nil
	repo.listAdminsPage = &adminListPage{
		Items: []AdminOperatorItem{},
		Next:  &adminCursor{GrantedAt: time.Now().UTC(), UserID: "user_alice"},
	}
	resp, apiErr, err := svc.ListAdmins(ctx, AdminRbacAdminsListRequest{})
	if apiErr != nil || err != nil || resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("ListAdmins next cursor: resp=%#v apiErr=%v err=%v", resp, apiErr, err)
	}
}

func TestServiceUpdateRoleErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)
	role, err := repo.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "cov_update_role"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	// 空白新名称 → 参数错误。
	blank := "   "
	if apiErr, err := svc.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &blank}); err != nil || apiErr == nil {
		t.Fatalf("blank name: apiErr=%v err=%v", apiErr, err)
	}

	repo.getRoleErr = errCovAdmin
	if _, err := svc.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: role.RoleID}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get role error: %v", err)
	}
	repo.getRoleErr = nil

	name := "cov_update_role_new"
	repo.updateRoleErr = &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
	if apiErr, err := svc.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &name}); apiErr == nil || err != nil {
		t.Fatalf("update api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.updateRoleErr = errCovAdmin
	if _, err := svc.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &name}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("update raw error: %v", err)
	}
	repo.updateRoleErr = nil

	repo.auditErr = errCovAdmin
	if apiErr, err := svc.UpdateRole(ctx, AdminRbacRoleUpdateRequest{RoleID: role.RoleID, Name: &name}); apiErr != nil || err != nil {
		t.Fatalf("update with audit failure: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceDeleteRoleErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)
	role, err := repo.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "cov_delete_role"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	repo.getRoleErr = errCovAdmin
	if _, err := svc.DeleteRole(ctx, role.RoleID); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get role error: %v", err)
	}
	repo.getRoleErr = nil

	repo.deleteRoleErr = &httpx.APIError{Code: httpx.CodeConflict, Message: "conflict"}
	if apiErr, err := svc.DeleteRole(ctx, role.RoleID); apiErr == nil || err != nil {
		t.Fatalf("delete api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.deleteRoleErr = errCovAdmin
	if _, err := svc.DeleteRole(ctx, role.RoleID); !errors.Is(err, errCovAdmin) {
		t.Fatalf("delete raw error: %v", err)
	}
	repo.deleteRoleErr = nil

	repo.auditErr = errCovAdmin
	if apiErr, err := svc.DeleteRole(ctx, role.RoleID); apiErr != nil || err != nil {
		t.Fatalf("delete with audit failure: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceSetRolePermissionsErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)
	role, err := base.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "cov_perm_role"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	// 非法权限键 → 参数错误。
	if apiErr, err := svc.SetRolePermissions(ctx, AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{"bogus.permission"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid permission set: apiErr=%v err=%v", apiErr, err)
	}

	// 非超管授予通配符 → 越权拒绝。
	covGrantPermissions(t, base, "user_alice", "cov_limited_rbac", permissionRbacManage)
	if apiErr, err := svc.SetRolePermissions(adminCtx("user_alice"), AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{wildcardPermission}}); err != nil || apiErr == nil {
		t.Fatalf("wildcard escalation: apiErr=%v err=%v", apiErr, err)
	}

	repo.getRoleErr = errCovAdmin
	if _, err := svc.SetRolePermissions(ctx, AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{permissionTagsRead}}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get role error: %v", err)
	}
	repo.getRoleErr = nil

	repo.setRolePermissionsErr = &httpx.APIError{Code: httpx.CodeConflict, Message: "conflict"}
	if apiErr, err := svc.SetRolePermissions(ctx, AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{permissionTagsRead}}); apiErr == nil || err != nil {
		t.Fatalf("set permissions api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.setRolePermissionsErr = errCovAdmin
	if _, err := svc.SetRolePermissions(ctx, AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{permissionTagsRead}}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("set permissions raw error: %v", err)
	}
	repo.setRolePermissionsErr = nil

	repo.auditErr = errCovAdmin
	if apiErr, err := svc.SetRolePermissions(ctx, AdminRbacRoleSetPermissionsRequest{RoleID: role.RoleID, PermissionKeys: []string{permissionTagsRead}}); apiErr != nil || err != nil {
		t.Fatalf("set permissions with audit failure: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceGrantAdminErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)
	role, err := base.CreateRole(ctx, AdminRbacRoleCreateRequest{Name: "cov_grant_role"})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	if apiErr, err := svc.GrantAdmin(ctx, AdminRbacGrantAdminRequest{}); err != nil || apiErr == nil {
		t.Fatalf("empty user: apiErr=%v err=%v", apiErr, err)
	}

	covGrantPermissions(t, base, "user_alice", "cov_limited_grant", permissionRbacManage)
	if apiErr, err := svc.GrantAdmin(adminCtx("user_alice"), AdminRbacGrantAdminRequest{UserID: "user_bob", AwardAllPermissions: true}); err != nil || apiErr == nil {
		t.Fatalf("award all without superadmin: apiErr=%v err=%v", apiErr, err)
	}

	repo.grantAdminErr = &httpx.APIError{Code: httpx.CodeConflict, Message: "conflict"}
	if apiErr, err := svc.GrantAdmin(ctx, AdminRbacGrantAdminRequest{UserID: "user_bob", RoleIDs: []string{role.RoleID}}); apiErr == nil || err != nil {
		t.Fatalf("grant api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.grantAdminErr = errCovAdmin
	if _, err := svc.GrantAdmin(ctx, AdminRbacGrantAdminRequest{UserID: "user_bob", RoleIDs: []string{role.RoleID}}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("grant raw error: %v", err)
	}
	repo.grantAdminErr = nil

	repo.auditErr = errCovAdmin
	if apiErr, err := svc.GrantAdmin(ctx, AdminRbacGrantAdminRequest{UserID: "user_bob", RoleIDs: []string{role.RoleID}}); apiErr != nil || err != nil {
		t.Fatalf("grant with audit failure: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceRevokeAdminErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if apiErr, err := svc.RevokeAdmin(ctx, ""); err != nil || apiErr == nil {
		t.Fatalf("empty user: apiErr=%v err=%v", apiErr, err)
	}

	repo.getActorErrUser = "user_bob"
	repo.getActorErr = errCovAdmin
	if _, err := svc.RevokeAdmin(ctx, "user_bob"); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get actor error: %v", err)
	}
	repo.getActorErr = nil
	repo.getActorErrUser = ""

	// 低权限管理员撤销超管 → 受保护目标。
	covGrantPermissions(t, base, "user_alice", "cov_limited_revoke", permissionRbacManage)
	if apiErr, err := svc.RevokeAdmin(adminCtx("user_alice"), "admin"); err != nil || apiErr == nil {
		t.Fatalf("protected target: apiErr=%v err=%v", apiErr, err)
	}

	// 超管撤销自己 → 拒绝。
	if apiErr, err := svc.RevokeAdmin(ctx, "admin"); err != nil || apiErr == nil {
		t.Fatalf("self revoke: apiErr=%v err=%v", apiErr, err)
	}

	// user_bob 需要先是管理员才能撤销。
	if err := base.GrantAdmin(ctx, AdminGrantUpsert{UserID: "user_bob", RoleIDs: nil, GrantAllPermissions: false}); err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	repo.revokeAdminErr = &httpx.APIError{Code: httpx.CodeConflict, Message: "conflict"}
	if apiErr, err := svc.RevokeAdmin(ctx, "user_bob"); apiErr == nil || err != nil {
		t.Fatalf("revoke api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.revokeAdminErr = errCovAdmin
	if _, err := svc.RevokeAdmin(ctx, "user_bob"); !errors.Is(err, errCovAdmin) {
		t.Fatalf("revoke raw error: %v", err)
	}
	repo.revokeAdminErr = nil

	repo.auditErr = errCovAdmin
	if apiErr, err := svc.RevokeAdmin(ctx, "user_bob"); apiErr != nil || err != nil {
		t.Fatalf("revoke with audit failure: apiErr=%v err=%v", apiErr, err)
	}
}

// ── 用户 / 群组 ────────────────────────────────────────────────────────

func TestServiceSearchUsersErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if _, apiErr, err := svc.SearchUsers(ctx, AdminUserSearchRequest{Status: UserAccountStatus("bogus")}); err != nil || apiErr == nil {
		t.Fatalf("invalid status: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.SearchUsers(ctx, AdminUserSearchRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid cursor: apiErr=%v err=%v", apiErr, err)
	}

	repo.searchUsersErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.SearchUsers(ctx, AdminUserSearchRequest{Nickname: "Alice"}); apiErr == nil || err != nil {
		t.Fatalf("search api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchUsersErr = errCovAdmin
	if _, apiErr, err := svc.SearchUsers(ctx, AdminUserSearchRequest{Nickname: "Alice"}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("search raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchUsersErr = nil

	// 无 read_ip 权限：IP 字段脱敏，且分页游标被编码。
	repo.searchUsersPage = &adminUserPage{
		Items: []AdminUserItem{{User: UserInfo{UserID: "user_alice"}, RegisterIP: types.StrPtr("1.2.3.4"), LastLoginIP: types.StrPtr("5.6.7.8")}},
		Next:  &adminUserCursor{CreatedAt: time.Now().UTC(), UserID: "user_alice"},
	}
	covGrantPermissions(t, base, "user_alice", "cov_limited_users_read", permissionUsersRead)
	resp, apiErr, err := svc.SearchUsers(adminCtx("user_alice"), AdminUserSearchRequest{Nickname: "Alice"})
	if apiErr != nil || err != nil {
		t.Fatalf("limited search: apiErr=%v err=%v", apiErr, err)
	}
	if len(resp.Items) != 1 || resp.Items[0].RegisterIP != nil || resp.Items[0].LastLoginIP != nil {
		t.Fatalf("IP should be masked: %#v", resp.Items)
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("expected encoded next cursor: %#v", resp)
	}
}

func TestServiceOpenAdminUserErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if _, apiErr, err := svc.OpenAdminUser(ctx, "  "); err != nil || apiErr == nil {
		t.Fatalf("empty user id: apiErr=%v err=%v", apiErr, err)
	}
	repo.getUserErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.OpenAdminUser(ctx, "user_alice"); apiErr == nil || err != nil {
		t.Fatalf("get user api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.getUserErr = errCovAdmin
	if _, apiErr, err := svc.OpenAdminUser(ctx, "user_alice"); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("get user raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.getUserErr = nil

	covGrantPermissions(t, base, "user_alice", "cov_limited_users_open", permissionUsersRead)
	item, apiErr, err := svc.OpenAdminUser(adminCtx("user_alice"), "user_bob")
	if apiErr != nil || err != nil {
		t.Fatalf("limited open: apiErr=%v err=%v", apiErr, err)
	}
	if item.RegisterIP != nil || item.LastLoginIP != nil {
		t.Fatalf("IP should be masked: %#v", item)
	}
}

func TestServiceSetUserBanErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	// 解封参数非法 / 用户不存在。
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "  "}); err != nil || apiErr == nil {
		t.Fatalf("empty user: apiErr=%v err=%v", apiErr, err)
	}
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_nobody"}); err != nil || apiErr == nil {
		t.Fatalf("unknown user: apiErr=%v err=%v", apiErr, err)
	}

	repo.getUserErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get user error: %v", err)
	}
	repo.getUserErr = nil
	repo.userBanStateErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("ban state error: %v", err)
	}
	repo.userBanStateErr = nil

	// 未封禁用户解封是 no-op，但审计失败要上抛。
	repo.auditErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("no-op unban audit error: %v", err)
	}
	repo.auditErr = nil

	// 已封禁用户解封：取目标管理员资料失败。
	base.banStates["user_bob"] = UserBanState{Status: UserAccountStatusBanned, Reason: "spam"}
	repo.getActorErrUser = "user_bob"
	repo.getActorErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("target actor error: %v", err)
	}
	repo.getActorErr = nil
	repo.getActorErrUser = ""

	// 低权限管理员不能解封管理员目标（目标需处于封禁态才会走到保护判定）。
	base.banStates["admin"] = UserBanState{Status: UserAccountStatusBanned, Reason: "spam"}
	covGrantPermissions(t, base, "user_alice", "cov_limited_ban", permissionUsersWrite)
	if apiErr, err := svc.SetUserBan(adminCtx("user_alice"), AdminSetUserBanRequest{UserID: "admin"}); err != nil || apiErr == nil {
		t.Fatalf("protected target unban: apiErr=%v err=%v", apiErr, err)
	}

	// 未装配 accountStateWriter / sessionAdmin 时显式失败（封禁路径）。
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); err == nil {
		t.Fatal("expected account state writer missing error")
	}

	svc.SetAccountStateWriter(covBanWriter{err: errCovAdmin})
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("writer raw error: %v", err)
	}
	svc.SetAccountStateWriter(covBanWriter{err: &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}})
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob"}); apiErr == nil || err != nil {
		t.Fatalf("writer api error: apiErr=%v err=%v", apiErr, err)
	}
	svc.SetAccountStateWriter(testAccountStateWriter{store: base})

	// 会话管理员未装配 / 列会话失败（封禁路径）。
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); err == nil {
		t.Fatal("expected session admin missing error")
	}
	svc.SetSessionAdmin(covSessionAdmin{listErr: errCovAdmin})
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("list sessions error: %v", err)
	}

	// 单条会话撤销失败降级为 Warn，不影响封禁成功。
	svc.SetSessionAdmin(covSessionAdmin{revokeErr: errCovAdmin})
	if apiErr, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); apiErr != nil || err != nil {
		t.Fatalf("revoke failure should not fail ban: apiErr=%v err=%v", apiErr, err)
	}

	// 封禁成功但审计写失败 → 上抛。
	svc.SetSessionAdmin(testSessionAdmin{store: base})
	repo.auditErr = errCovAdmin
	if _, err := svc.SetUserBan(ctx, AdminSetUserBanRequest{UserID: "user_bob", Active: true, Reason: "spam"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("ban audit error: %v", err)
	}
}

func TestServiceSearchGroupsErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if _, apiErr, err := svc.SearchGroups(ctx, AdminGroupSearchRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid cursor: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchGroupsErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.SearchGroups(ctx, AdminGroupSearchRequest{Name: "hot"}); apiErr == nil || err != nil {
		t.Fatalf("search api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchGroupsErr = errCovAdmin
	if _, apiErr, err := svc.SearchGroups(ctx, AdminGroupSearchRequest{Name: "hot"}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("search raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchGroupsErr = nil
	repo.searchGroupsPage = &adminGroupPage{
		Items: []AdminGroupItem{},
		Next:  &adminGroupCursor{CreatedAt: time.Now().UTC(), GroupID: "group_hotspot"},
	}
	resp, apiErr, err := svc.SearchGroups(ctx, AdminGroupSearchRequest{Name: "hot"})
	if apiErr != nil || err != nil || resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("next cursor: resp=%#v apiErr=%v err=%v", resp, apiErr, err)
	}
}

func TestServiceOpenAdminGroupErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if _, apiErr, err := svc.OpenAdminGroup(ctx, "  "); err != nil || apiErr == nil {
		t.Fatalf("empty group id: apiErr=%v err=%v", apiErr, err)
	}
	repo.getGroupErr = errCovAdmin
	if _, apiErr, err := svc.OpenAdminGroup(ctx, "group_hotspot"); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("get group error: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceDismissGroupErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := NewInMemoryRepository()

	// 未装配 groupModerator。
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)
	if apiErr, err := svc.DismissGroup(ctx, AdminDismissGroupRequest{GroupID: "group_hotspot"}); err == nil || apiErr != nil {
		t.Fatalf("missing moderator: apiErr=%v err=%v", apiErr, err)
	}
	if apiErr, err := svc.DismissGroup(ctx, AdminDismissGroupRequest{GroupID: "  "}); err != nil || apiErr == nil {
		t.Fatalf("empty group: apiErr=%v err=%v", apiErr, err)
	}

	svc.SetGroupModerator(covGroupModerator{err: groupsmodule.ErrGroupNotFound})
	if apiErr, err := svc.DismissGroup(ctx, AdminDismissGroupRequest{GroupID: "group_missing"}); err != nil || apiErr == nil {
		t.Fatalf("group not found: apiErr=%v err=%v", apiErr, err)
	}
	svc.SetGroupModerator(covGroupModerator{err: &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}})
	if apiErr, err := svc.DismissGroup(ctx, AdminDismissGroupRequest{GroupID: "group_hotspot"}); apiErr == nil || err != nil {
		t.Fatalf("moderator api error: apiErr=%v err=%v", apiErr, err)
	}
	svc.SetGroupModerator(covGroupModerator{err: errCovAdmin})
	if _, err := svc.DismissGroup(ctx, AdminDismissGroupRequest{GroupID: "group_hotspot"}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("moderator raw error: %v", err)
	}
}

var _ = observability.L
var _ = outbox.TopicSystemAnnouncement
var _ = strings.TrimSpace
