package admin

import (
	"context"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// --- stub provider ---

type stubProvider struct {
	config AppConfig
}

func (p *stubProvider) GetAppConfig(_ context.Context) (AppConfig, error) {
	return p.config, nil
}

func (p *stubProvider) UpdateAppConfig(_ context.Context, _ string, apply func(AppConfig) (AppConfig, *httpx.APIError)) (AppConfig, *httpx.APIError, error) {
	updated, apiErr := apply(p.config)
	if apiErr != nil {
		return AppConfig{}, apiErr, nil
	}
	p.config = updated
	return updated, nil, nil
}

// --- stub push service ---

type stubPushService struct {
	announced bool
	pushed    bool
}

func (s *stubPushService) AppendSystemAnnouncement(_ context.Context, _ []string, _ systemmodule.SystemAnnouncementItem) error {
	s.announced = true
	return nil
}

func (s *stubPushService) PushSystemAnnouncement(_ systemmodule.SystemAnnouncementItem) {
	s.pushed = true
}

// --- helpers ---

func adminCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func defaultConfig() AppConfig {
	return AppConfig{}
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
	if apiErr == nil || apiErr.Code != 10100 {
		t.Fatalf("expected 10100, got %v", apiErr)
	}
}

func TestServiceGetMeRejectsNonAdmin(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, apiErr, err := svc.GetMe(adminCtx("nobody"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10200 {
		t.Fatalf("expected 10200, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10200 {
		t.Fatalf("expected 10200, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10200 {
		t.Fatalf("expected 10200, got %v", apiErr)
	}
}

func TestServiceCreateRoleRejectsEmptyName(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	_, apiErr, err := svc.CreateRole(adminCtx("admin1"), AdminRbacRoleCreateRequest{Name: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
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
	apiErr, err := svc.GrantAdmin(adminCtx("admin1"), AdminRbacGrantAdminRequest{UserID: "u2", GrantAllPermissions: true})
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
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10200 {
		t.Fatalf("expected 10200, got %v", apiErr)
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

// --- GetAuditLog ---

func TestServiceGetAuditLogRejectsEmptyID(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	_, apiErr, err := svc.GetAuditLog(adminCtx("admin1"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
	}
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

// --- UpdateFeatures ---

func TestServiceUpdateFeaturesHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	enabled := true
	apiErr, err := svc.UpdateFeatures(adminCtx("admin1"), UpdateFeaturesRequest{EnableMoments: &enabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr != nil {
		t.Fatalf("unexpected api error: %v", apiErr)
	}
}

// --- UpdateRateLimits ---

func TestServiceUpdateRateLimitsRejectsZero(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	zero := 0
	apiErr, err := svc.UpdateRateLimits(adminCtx("admin1"), UpdateRateLimitsRequest{RateLimits: PartialRateLimitConfig{GlobalPerHour: &zero}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
	}
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
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
	}
}

// --- BanUser ---

func TestServiceBanUserRejectsSelf(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.BanUser(adminCtx("admin1"), AdminBanUserRequest{UserID: "admin1", Reason: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10400 {
		t.Fatalf("expected 10400, got %v", apiErr)
	}
}

// --- UnbanUser ---

func TestServiceUnbanUserRejectsNotBanned(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.UnbanUser(adminCtx("admin1"), AdminUnbanUserRequest{UserID: "u2", Reason: "oops"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300 (user not found or not banned), got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
	}
}

// --- SearchReports ---

func TestServiceSearchReportsHappyPath(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	result, apiErr, err := svc.SearchReports(adminCtx("admin1"), AdminReportSearchRequest{Pagination: &PaginationParams{Limit: 10}})
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
	apiErr, err := svc.HandleReport(adminCtx("admin1"), AdminReportHandleRequest{ReportID: "r1", Action: "invalid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
	}
}

// --- DeleteTag ---

func TestServiceDeleteTagRejectsEmptyName(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteTag(adminCtx("admin1"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
	}
}

func TestServiceDeleteTagRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteTag(adminCtx("admin1"), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
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
	if apiErr == nil || apiErr.Code != 10000 {
		t.Fatalf("expected 10000, got %v", apiErr)
	}
}

func TestServiceDeleteAnnouncementRejectsNotFound(t *testing.T) {
	svc, repo, _, _ := newTestService()
	repo.GrantAdmin(context.Background(), AdminGrantUpsert{UserID: "admin1", GrantAllPermissions: true})
	apiErr, err := svc.DeleteAnnouncement(adminCtx("admin1"), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr == nil || apiErr.Code != 10300 {
		t.Fatalf("expected 10300, got %v", apiErr)
	}
}
