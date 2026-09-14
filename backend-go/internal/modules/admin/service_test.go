package admin

import (
	"context"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// --- stub provider ---

type stubProvider struct {
	config AdminAppConfig
}

func (p *stubProvider) GetAppConfig(_ context.Context) (AdminAppConfig, error) {
	return p.config, nil
}

func (p *stubProvider) UpdateAppConfig(_ context.Context, _ string, apply func(AdminAppConfig) (AdminAppConfig, *httpx.APIError)) (AdminAppConfig, *httpx.APIError, error) {
	updated, apiErr := apply(p.config)
	if apiErr != nil {
		return AdminAppConfig{}, apiErr, nil
	}
	p.config = updated
	return updated, nil, nil
}

// --- stub push service ---

type stubPushService struct {
	announced   bool
	announcedID string
	pushed      bool
	pushedID    string
}

func (s *stubPushService) AppendSystemAnnouncement(_ context.Context, _ []string, _ systemmodule.SystemAnnouncementItem) (string, error) {
	s.announced = true
	s.announcedID = "notif_ann_1"
	return s.announcedID, nil
}

func (s *stubPushService) NewAnnouncementNotifyID() string {
	s.announced = true
	s.announcedID = "notif_ann_1"
	return s.announcedID
}

func (s *stubPushService) AppendSystemAnnouncementWithID(_ context.Context, id string, _ []string, _ systemmodule.SystemAnnouncementItem) (string, error) {
	s.announced = true
	s.announcedID = id
	return id, nil
}

func (s *stubPushService) PushSystemAnnouncement(_ systemmodule.SystemAnnouncementItem, notificationID string) {
	s.pushed = true
	s.pushedID = notificationID
}

func (s *stubPushService) AppendReportResolved(_ context.Context, _ string, _ string, _ string, _ string, _ string) (string, error) {
	return "notif_report_1", nil
}

func (s *stubPushService) AppendReportDismissed(_ context.Context, _ string, _ string, _ string, _ string, _ string) (string, error) {
	return "notif_report_2", nil
}

// --- helpers ---

func adminCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func defaultConfig() AdminAppConfig {
	return AdminAppConfig{}
}

func newTestService() (*Service, *InMemoryRepository, *stubProvider, *stubPushService) {
	repo := NewInMemoryRepository()
	provider := &stubProvider{config: defaultConfig()}
	push := &stubPushService{}
	mods := NewInMemoryModerationStore()
	svc := NewService(repo, provider, mods, push)
	return svc, repo, provider, push
}

// --- GetMe ---

func TestServiceGetMeRejectsNoAuth(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.GetMe(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeUnauthorized {
		t.Fatalf("expected httpx.CodeUnauthorized, got %v", apiErr)
	}
}

func TestServiceGetMeRejectsNonAdmin(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.GetMe(adminCtx("nobody"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

func TestServiceGetMeReturnsAdmin(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	perms, apiErr, err := svc.GetMe(adminCtx("admin1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	if len(perms.Permissions) == 0 {
		t.Fatal("expected permissions")
	}
}

// --- ListPermissions ---

func TestServiceListPermissionsRequiresAdmin(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.ListPermissions(adminCtx("nobody"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

func TestServiceListPermissionsHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.ListPermissions(adminCtx("admin1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected permissions")
	}
}

// --- CreateRole ---

func TestServiceCreateRoleRequiresPermission(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", RoleIDs: []string{"role_with_users_read"}})
	_, apiErr, err := svc.CreateRole(adminCtx("admin1"), AdminRbacRoleCreateRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

func TestServiceCreateRoleRejectsEmptyName(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	_, apiErr, err := svc.CreateRole(adminCtx("admin1"), AdminRbacRoleCreateRequest{Name: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

func TestServiceCreateRoleHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	item, apiErr, err := svc.CreateRole(adminCtx("admin1"), AdminRbacRoleCreateRequest{Name: "moderator"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	if item.Name != "moderator" {
		t.Fatalf("expected moderator, got %s", item.Name)
	}
}

// --- UpdateRole ---

func TestServiceUpdateRoleRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.UpdateRole(adminCtx("admin1"), AdminRbacRoleUpdateRequest{RoleID: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// --- DeleteRole ---

func TestServiceDeleteRoleRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteRole(adminCtx("admin1"), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

func TestServiceDeleteRoleHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	item, _, _ := svc.CreateRole(adminCtx("admin1"), AdminRbacRoleCreateRequest{Name: "toremove"})
	apiErr, err := svc.DeleteRole(adminCtx("admin1"), item.RoleID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
}

// --- GrantAdmin / RevokeAdmin ---

func TestServiceGrantAdminRejectsNoRoles(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.GrantAdmin(adminCtx("admin1"), AdminRbacGrantAdminRequest{UserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil {
		t.Fatal("expected error for no roles and no grant_all")
	}
}

func TestServiceGrantAdminHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.GrantAdmin(adminCtx("admin1"), AdminRbacGrantAdminRequest{UserID: "u2", AwardAllPermissions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
}

func TestServiceRevokeAdminRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.RevokeAdmin(adminCtx("admin1"), "nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// --- ListAuditLogs ---

func TestServiceListAuditLogsRequiresPermission(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", RoleIDs: []string{"role_with_audit"}})
	_, apiErr, err := svc.ListAuditLogs(adminCtx("admin1"), AuditLogListRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

func TestServiceListAuditLogsHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.ListAuditLogs(adminCtx("admin1"), AuditLogListRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_ = result
}

// --- GetAppConfig ---

func TestServiceGetAppConfigHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	cfg, apiErr, err := svc.GetAppConfig(adminCtx("admin1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_ = cfg
}

// --- SearchUsers ---

func TestServiceSearchUsersHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.SearchUsers(adminCtx("admin1"), AdminUserSearchRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_ = result
}

// --- OpenAdminUser ---

func TestServiceOpenAdminUserRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	_, apiErr, err := svc.OpenAdminUser(adminCtx("admin1"), "nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// --- BanUser ---

func TestServiceBanUserRejectsSelf(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "admin1", Reason: "test", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeConflict {
		t.Fatalf("expected httpx.CodeConflict, got %v", apiErr)
	}
}

// --- SetUserBan（unban 路径）---

func TestServiceUnbanUserRejectsUnknownUser(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "u2", Reason: "oops", Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound (user not found), got %v", apiErr)
	}
}

// 契约：解封未封禁用户视为 no-op 成功（设状态操作重复达同状态返回成功，见 api_spec_errors.md）
func TestServiceUnbanUserNoopWhenNotBanned(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_alice", Reason: "oops", Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("expected no-op success, got %v", apiErr)
	}
}

// --- SearchGroups ---

func TestServiceSearchGroupsHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.SearchGroups(adminCtx("admin1"), AdminGroupSearchRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_ = result
}

// --- OpenAdminGroup ---

func TestServiceOpenAdminGroupRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	_, apiErr, err := svc.OpenAdminGroup(adminCtx("admin1"), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// --- SearchReports ---

func TestServiceSearchReportsHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.SearchReports(adminCtx("admin1"), AdminReportListRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_ = result
}

// --- HandleReport ---

func TestServiceHandleReportRejectsInvalidAction(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.HandleReport(adminCtx("admin1"), JudgeAdminReportRequest{ReportID: "r1", Action: "invalid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

// --- SetTagBan ---

func TestServiceSetTagBanRejectsEmptyName(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetTagBan(adminCtx("admin1"), AdminSetTagBanRequest{Name: "", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

func TestServiceSetTagBanRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetTagBan(adminCtx("admin1"), AdminSetTagBanRequest{Name: "nonexistent", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// TestServiceSetTagBanRejectsUnbanWithHardDelete：解屏蔽与物理删除语义冲突。
func TestServiceSetTagBanRejectsUnbanWithHardDelete(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetTagBan(adminCtx("admin1"), AdminSetTagBanRequest{Name: "hotspot", Active: false, HardDelete: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

// TestServiceSetTagBanUnbanNoOp：对未屏蔽标签解屏蔽 no-op 成功。
func TestServiceSetTagBanUnbanNoOp(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetTagBan(adminCtx("admin1"), AdminSetTagBanRequest{Name: "never_blocked", Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("expected no-op success, got %v", apiErr)
	}
}

// --- DeleteAnnouncement ---

func TestServiceDeleteAnnouncementRejectsEmptyID(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteAnnouncement(adminCtx("admin1"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

func TestServiceDeleteAnnouncementRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteAnnouncement(adminCtx("admin1"), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeNotFound {
		t.Fatalf("expected httpx.CodeNotFound, got %v", apiErr)
	}
}

// --- SearchUsers (no auth) ---

func TestServiceSearchUsersRejectsNoAuth(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.SearchUsers(context.Background(), AdminUserSearchRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeUnauthorized {
		t.Fatalf("expected httpx.CodeUnauthorized, got %v", apiErr)
	}
}

// --- BanUser happy path ---

func TestServiceBanUserHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_bob", Reason: "violation", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	banned, checkErr := repo.IsBanned(context.Background(), "user_bob")
	if checkErr != nil {
		t.Fatalf("unexpected error checking ban state: %v", checkErr)
	}
	if !banned {
		t.Fatal("expected user_bob to be banned")
	}
}

// --- BanUser rejects empty userID ---

func TestServiceBanUserRejectsEmptyUserID(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "", Reason: "test", Active: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected httpx.CodeInvalidRequest, got %v", apiErr)
	}
}

// --- UnbanUser happy path ---

func TestServiceUnbanUserHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	// 先封禁用户
	if apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_bob", Reason: "violation", Active: true}); err != nil {
		t.Fatalf("ban setup error: %v", err)
	} else if apiErr != nil {
		t.Fatalf("ban setup api error: %v", apiErr)
	}
	// 解封
	apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_bob", Reason: "appeal approved", Active: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	banned, checkErr := repo.IsBanned(context.Background(), "user_bob")
	if checkErr != nil {
		t.Fatalf("unexpected error checking ban state: %v", checkErr)
	}
	if banned {
		t.Fatal("expected user_bob to be unbanned")
	}
}

// --- SearchGroups returns results ---

func TestServiceSearchGroupsReturnsResults(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.SearchGroups(adminCtx("admin1"), AdminGroupSearchRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected at least one group in results")
	}
}

// --- DismissGroup happy path ---

func TestServiceDismissGroupHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DismissGroup(adminCtx("admin1"), AdminDismissGroupRequest{GroupID: "group_hotspot", Reason: "policy violation"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
	_, found, _ := repo.GetGroup(context.Background(), "group_hotspot")
	if found {
		t.Fatal("expected group_hotspot to be dismissed (not found)")
	}
}

// --- SearchReports rejects no permission ---

func TestServiceSearchReportsRejectsNoPermission(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", RoleIDs: []string{"role_with_users_read"}})
	_, apiErr, err := svc.SearchReports(adminCtx("admin1"), AdminReportListRequest{Pagination: &PaginationParams{Limit: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected httpx.CodeForbidden, got %v", apiErr)
	}
}

// --- HandleReport happy path ---

func TestServiceHandleReportHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.HandleReport(adminCtx("admin1"), JudgeAdminReportRequest{
		ReportID: "report_1",
		Action:   ReportActionDismiss,
		Note:     "false alarm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
}

// --- IsBanned 限期封禁到期判定 ---

func TestIsBannedHonorsBanExpiry(t *testing.T) {
	svc, repo, _, _ := newTestService()
	ctx := context.Background()
	repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	seven := 7
	if apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_bob", Reason: "violation", Active: true, DurationDays: &seven}); err != nil || apiErr != nil {
		t.Fatalf("ban setup failed: apiErr=%v err=%v", apiErr, err)
	}
	// 未到期仍封禁。
	if banned, err := repo.IsBanned(ctx, "user_bob"); err != nil || !banned {
		t.Fatalf("expected active temporary ban, banned=%v err=%v", banned, err)
	}
	// 回拨 BannedAt 到 8 天前（7 天限期已过）→ 到期视为解封。
	st := repo.banStates["user_bob"]
	st.BannedAt = time.Now().UTC().Add(-8 * 24 * time.Hour)
	repo.banStates["user_bob"] = st
	if banned, err := repo.IsBanned(ctx, "user_bob"); err != nil || banned {
		t.Fatalf("expected expired ban to read as unbanned, banned=%v err=%v", banned, err)
	}
	// 永久封禁不受时间流逝影响（fail-closed）。
	if apiErr, err := svc.SetUserBan(adminCtx("admin1"), AdminSetUserBanRequest{UserID: "user_alice", Reason: "violation", Active: true}); err != nil || apiErr != nil {
		t.Fatalf("permanent ban setup failed: apiErr=%v err=%v", apiErr, err)
	}
	perm := repo.banStates["user_alice"]
	perm.BannedAt = time.Now().UTC().Add(-365 * 24 * time.Hour)
	repo.banStates["user_alice"] = perm
	if banned, err := repo.IsBanned(ctx, "user_alice"); err != nil || !banned {
		t.Fatalf("expected permanent ban to persist, banned=%v err=%v", banned, err)
	}
}

func TestUserBanStateExpired(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name  string
		state UserBanState
		want  bool
	}{
		{"未封禁", UserBanState{Status: UserAccountStatusNormal}, false},
		{"限期内", UserBanState{Status: UserAccountStatusBanned, DurationDays: 7, BannedAt: now.Add(-24 * time.Hour)}, false},
		{"已到期", UserBanState{Status: UserAccountStatusBanned, DurationDays: 7, BannedAt: now.Add(-8 * 24 * time.Hour)}, true},
		{"永久封禁", UserBanState{Status: UserAccountStatusBanned, BannedAt: now.Add(-365 * 24 * time.Hour)}, false},
		{"缺时间戳按永久处理", UserBanState{Status: UserAccountStatusBanned, DurationDays: 7}, false},
	}
	for _, tc := range cases {
		if got := tc.state.Expired(now); got != tc.want {
			t.Fatalf("%s: Expired=%v, want %v", tc.name, got, tc.want)
		}
	}
}
