package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

type Service struct {
	repo            Repository
	provider        Provider
	moderationStore ModerationStore
	pushService     announcementPushService
}

type announcementPushService interface {
	PushSystemAnnouncement(userID, announcementID, severity string)
}

func NewService(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Service {
	if moderationStore == nil {
		moderationStore = NewInMemoryModerationStore()
	}
	return &Service{repo: repo, provider: provider, moderationStore: moderationStore, pushService: pushService}
}

func (s *Service) actorFromContext(ctx context.Context) (AdminActor, bool, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return AdminActor{}, false, &httpx.APIError{Code: 10100, Message: "Access Token 无效或过期"}
	}
	actor, found, err := s.repo.GetActor(ctx, principal.UserID)
	if err != nil {
		return AdminActor{}, false, err
	}
	return actor, found, nil
}

func (s *Service) GetMe(ctx context.Context) (AdminPermissions, *httpx.APIError, error) {
	actor, found, err := s.actorFromContext(ctx)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return AdminPermissions{}, apiErr, nil
		}
		return AdminPermissions{}, nil, err
	}
	if !found {
		return AdminPermissions{}, &httpx.APIError{Code: 10200, Message: "非系统管理员"}, nil
	}
	return actor.Response(), nil, nil
}

func (s *Service) ListPermissions(ctx context.Context) (AdminPermissionList, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return AdminPermissionList{}, apiErr, err
	}
	items, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return AdminPermissionList{}, nil, err
	}
	return AdminPermissionList{Items: items}, nil, nil
}

func (s *Service) ListRoles(ctx context.Context) (AdminRoleList, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return AdminRoleList{}, apiErr, err
	}
	items, err := s.repo.ListRoles(ctx)
	if err != nil {
		return AdminRoleList{}, nil, err
	}
	return AdminRoleList{Items: items}, nil, nil
}

func (s *Service) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return AdminRoleItem{}, apiErr, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return AdminRoleItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, err := s.repo.CreateRole(ctx, req)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return AdminRoleItem{}, apiErr, nil
		}
		return AdminRoleItem{}, nil, err
	}
	return item, nil, nil
}

func (s *Service) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) (*httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.RoleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.repo.UpdateRole(ctx, req); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) DeleteRole(ctx context.Context, roleID string) (*httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return apiErr, err
	}
	if roleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.repo.DeleteRole(ctx, roleID); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) SetRolePermissions(ctx context.Context, req AdminRbacRoleSetPermissionsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.RoleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	permissionKeys := sortAndUniqueStrings(req.PermissionKeys)
	if !validPermissionSet(permissionKeys) {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if containsString(permissionKeys, wildcardPermission) && !actor.IsSuperAdmin() {
		return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.rbac.manage"}, nil
	}
	if err := s.repo.SetRolePermissions(ctx, req.RoleID, permissionKeys); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) ListAdmins(ctx context.Context, req AdminRbacAdminsListRequest) (PaginatedAdminOperators, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		return PaginatedAdminOperators{}, apiErr, err
	}
	pagination := PaginationParams{}
	if req.Pagination != nil {
		pagination = *req.Pagination
	}
	page, err := s.repo.ListAdmins(ctx, pagination)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAdminOperators{}, apiErr, nil
		}
		return PaginatedAdminOperators{}, nil, err
	}
	resp := PaginatedAdminOperators{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAdminCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) GrantAdmin(ctx context.Context, req AdminRbacGrantAdminRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.UserID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.GrantAllPermissions && len(req.RoleIDs) > 0 {
		return httpx.InvalidRequest("grant_all_permissions 与 role_ids 互斥"), nil
	}
	if req.GrantAllPermissions && !actor.IsSuperAdmin() {
		return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.rbac.manage"}, nil
	}
	roleIDs := sortAndUniqueStrings(req.RoleIDs)
	if !req.GrantAllPermissions && len(roleIDs) == 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: req.UserID, GrantAllPermissions: req.GrantAllPermissions, RoleIDs: roleIDs}); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) RevokeAdmin(ctx context.Context, userID string) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if userID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	target, found, err := s.repo.GetActor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "管理员不存在"}, nil
	}
	if target.IsSuperAdmin() && !actor.IsSuperAdmin() {
		return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.rbac.manage"}, nil
	}
	if err := s.repo.RevokeAdmin(ctx, userID); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) GetAppConfig(ctx context.Context) (AppConfig, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAppConfigRead); apiErr != nil || err != nil {
		return AppConfig{}, apiErr, err
	}
	cfg, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return AppConfig{}, nil, err
	}
	return cfg, nil, nil
}

func (s *Service) SearchTags(ctx context.Context, req AdminTagSearchRequest) (PaginatedAdminTags, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionTagsRead); apiErr != nil || err != nil {
		return PaginatedAdminTags{}, apiErr, err
	}
	if apiErr := validateTagSearchRequest(req); apiErr != nil {
		return PaginatedAdminTags{}, apiErr, nil
	}
	pagination := PaginationParams{}
	if req.Pagination != nil {
		pagination = *req.Pagination
	}
	cursor, apiErr := decodeTagCursor(pagination.Cursor)
	if apiErr != nil {
		return PaginatedAdminTags{}, apiErr, nil
	}
	var cursorPtr *tagCursor
	if pagination.Cursor != "" {
		cursorPtr = &cursor
	}
	page, err := s.moderationStore.SearchTags(ctx, TagSearchFilter{Keyword: req.Keyword, Limit: pagination.Limit, Cursor: cursorPtr})
	if err != nil {
		return PaginatedAdminTags{}, nil, err
	}
	resp := PaginatedAdminTags{Items: make([]AdminTagItem, 0, len(page.Items))}
	for _, item := range page.Items {
		resp.Items = append(resp.Items, AdminTagItem{Name: item.Name, UsageCount: item.UsageCount, Trend: defaultTagTrend, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	if page.Next != nil {
		resp.NextCursor = encodeTagCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) DeleteTag(ctx context.Context, name string) (*httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionTagsWrite); apiErr != nil || err != nil {
		return apiErr, err
	}
	if normalizeTagName(name) == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.moderationStore.DeleteTag(ctx, name); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) SearchAnnouncements(ctx context.Context, req AdminAnnouncementSearchRequest) (PaginatedAdminAnnouncements, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsRead); apiErr != nil || err != nil {
		return PaginatedAdminAnnouncements{}, apiErr, err
	}
	pagination := PaginationParams{}
	if req.Pagination != nil {
		pagination = *req.Pagination
	}
	cursor, apiErr := decodeAnnouncementCursor(pagination.Cursor)
	if apiErr != nil {
		return PaginatedAdminAnnouncements{}, apiErr, nil
	}
	var cursorPtr *systemmodule.AnnouncementCursor
	if pagination.Cursor != "" {
		cursorPtr = &systemmodule.AnnouncementCursor{CreatedAt: cursor.CreatedAt, AnnouncementID: cursor.AnnouncementID}
	}
	page, err := s.moderationStore.SearchAnnouncements(ctx, systemmodule.AnnouncementSearchFilter{Severity: req.Severity, Limit: pagination.Limit, Cursor: cursorPtr})
	if err != nil {
		return PaginatedAdminAnnouncements{}, nil, err
	}
	resp := PaginatedAdminAnnouncements{Items: make([]AdminAnnouncementItem, 0, len(page.Items))}
	for _, item := range page.Items {
		resp.Items = append(resp.Items, toAdminAnnouncementItem(item))
	}
	if page.Next != nil {
		resp.NextCursor = encodeAnnouncementCursor(announcementCursor{CreatedAt: page.Next.CreatedAt, AnnouncementID: page.Next.AnnouncementID})
	}
	return resp, nil, nil
}

func (s *Service) CreateAnnouncement(ctx context.Context, req CreateAnnouncementRequest) (AnnouncementCreated, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite)
	if apiErr != nil || err != nil {
		return AnnouncementCreated{}, apiErr, err
	}
	if apiErr := validateAnnouncementCreate(req); apiErr != nil {
		return AnnouncementCreated{}, apiErr, nil
	}
	input := systemmodule.AnnouncementCreateInput{
		Actor: systemmodule.AnnouncementActor{UserID: actor.UserID, Nickname: actor.UserID},
		Title: strings.TrimSpace(req.Title), Content: strings.TrimSpace(req.Content), Severity: req.Severity,
		ActionURL: strings.TrimSpace(req.ActionURL), SendPush: req.SendPush,
	}
	item, err := s.moderationStore.CreateAnnouncement(ctx, input)
	if err != nil {
		return AnnouncementCreated{}, nil, err
	}
	if req.SendPush && s.pushService != nil {
		s.pushService.PushSystemAnnouncement(actor.UserID, item.AnnouncementID, string(item.Severity))
	}
	return AnnouncementCreated{AnnouncementID: item.AnnouncementID}, nil, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateAnnouncementRequest) (*httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite); apiErr != nil || err != nil {
		return apiErr, err
	}
	if apiErr := validateAnnouncementUpdate(req); apiErr != nil {
		return apiErr, nil
	}
	_, err := s.moderationStore.UpdateAnnouncement(ctx, systemmodule.AnnouncementUpdateInput{
		AnnouncementID: req.AnnouncementID,
		Title:          trimOptional(req.Title),
		Content:        trimOptional(req.Content),
		Severity:       req.Severity,
		ActionURL:      trimOptional(req.ActionURL),
	})
	if err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) || err.Error() == "announcement not found" {
			return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) DeleteAnnouncement(ctx context.Context, announcementID string) (*httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite); apiErr != nil || err != nil {
		return apiErr, err
	}
	if strings.TrimSpace(announcementID) == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.moderationStore.DeleteAnnouncement(ctx, announcementID); err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) || err.Error() == "announcement not found" {
			return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
		}
		return nil, err
	}
	return nil, nil
}

func (s *Service) requirePermission(ctx context.Context, permission string) (AdminActor, *httpx.APIError, error) {
	actor, found, err := s.actorFromContext(ctx)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return AdminActor{}, apiErr, nil
		}
		return AdminActor{}, nil, err
	}
	if !found {
		return AdminActor{}, &httpx.APIError{Code: 10200, Message: "非系统管理员"}, nil
	}
	if !actor.HasPermission(permission) {
		return AdminActor{}, &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: " + permission}, nil
	}
	return actor, nil, nil
}
