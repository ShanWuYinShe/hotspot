package admin

import (
	"context"
	"strings"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
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
