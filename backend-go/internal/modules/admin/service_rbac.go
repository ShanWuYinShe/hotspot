package admin

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

func (s *Service) GetMe(ctx context.Context) (AdminPermissions, *httpx.APIError, error) {
	startedAt := time.Now()
	observability.L().Info("admin.get_me started")
	actor, found, err := s.actorFromContext(ctx)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			observability.L().Warn("admin.get_me auth failed", "error", apiErr, "elapsed_ms", time.Since(startedAt).Milliseconds())
			return AdminPermissions{}, apiErr, nil
		}
		observability.L().Error("admin.get_me error", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminPermissions{}, nil, err
	}
	if !found {
		observability.L().Warn("admin.get_me not admin", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminPermissions{}, httpx.Forbidden(httpx.ReasonNotAdmin, "非平台管理员"), nil
	}
	observability.L().Info("admin.get_me completed", "user_id", actor.UserID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return actor.Response(), nil, nil
}

func (s *Service) ListPermissions(ctx context.Context) (AdminPermissionList, *httpx.APIError, error) {
	startedAt := time.Now()
	observability.L().Info("admin.list_permissions started")
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		observability.L().Warn("admin.list_permissions permission denied", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminPermissionList{}, apiErr, err
	}
	items, err := s.repo.ListPermissions(ctx)
	if err != nil {
		observability.L().Error("admin.list_permissions query failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminPermissionList{}, nil, err
	}
	observability.L().Info("admin.list_permissions completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return AdminPermissionList{Items: items}, nil, nil
}

func (s *Service) ListRoles(ctx context.Context) (AdminRoleList, *httpx.APIError, error) {
	startedAt := time.Now()
	observability.L().Info("admin.list_roles started")
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		observability.L().Warn("admin.list_roles permission denied", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminRoleList{}, apiErr, err
	}
	items, err := s.repo.ListRoles(ctx)
	if err != nil {
		observability.L().Error("admin.list_roles query failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminRoleList{}, nil, err
	}
	observability.L().Info("admin.list_roles completed", "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return AdminRoleList{Items: items}, nil, nil
}

func (s *Service) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, *httpx.APIError, error) {
	startedAt := time.Now()
	observability.L().Info("admin.create_role started", "role_name", req.Name)
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		observability.L().Warn("admin.create_role permission denied", "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminRoleItem{}, apiErr, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return AdminRoleItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, err := s.repo.CreateRole(ctx, req)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			observability.L().Warn("admin.create_role business error", "error", apiErr, "elapsed_ms", time.Since(startedAt).Milliseconds())
			return AdminRoleItem{}, apiErr, nil
		}
		observability.L().Error("admin.create_role failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return AdminRoleItem{}, nil, err
	}
	_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionRoleCreate,
		TargetType: AuditTargetTypeRole,
		TargetID:   item.RoleID,
		TargetName: item.Name,
		Details: AuditDetails{
			"type":        string(AuditActionRoleCreate),
			"role_name":   item.Name,
			"permissions": item.Permissions,
		},
	}))
	return item, nil, nil
}

func (s *Service) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.RoleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	before, found, err := s.findRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
	}
	if err := s.repo.UpdateRole(ctx, req); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	after, _, err := s.findRole(ctx, req.RoleID)
	if err == nil {
		_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
			Action:     AuditActionRoleUpdate,
			TargetType: AuditTargetTypeRole,
			TargetID:   after.RoleID,
			TargetName: after.Name,
			Details: AuditDetails{
				"type":                string(AuditActionRoleUpdate),
				"role_name":           after.Name,
				"added_permissions":   diffStrings(after.Permissions, before.Permissions),
				"removed_permissions": diffStrings(before.Permissions, after.Permissions),
			},
		}))
	}
	return nil, nil
}

func (s *Service) DeleteRole(ctx context.Context, roleID string) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if roleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	before, found, err := s.findRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
	}
	if err := s.repo.DeleteRole(ctx, roleID); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionRoleDelete,
		TargetType: AuditTargetTypeRole,
		TargetID:   before.RoleID,
		TargetName: before.Name,
		Details: AuditDetails{
			"type":      string(AuditActionRoleDelete),
			"role_name": before.Name,
		},
	}))
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
		return httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permissionRbacManage), nil
	}
	before, found, err := s.findRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
	}
	if err := s.repo.SetRolePermissions(ctx, req.RoleID, permissionKeys); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionRoleSetPermissions,
		TargetType: AuditTargetTypeRole,
		TargetID:   before.RoleID,
		TargetName: before.Name,
		Details: AuditDetails{
			"type":        string(AuditActionRoleSetPermissions),
			"role_name":   before.Name,
			"permissions": permissionKeys,
		},
	}))
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
		if apiErr := asAPIError(err); apiErr != nil {
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
		return httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permissionRbacManage), nil
	}
	roleIDs := sortAndUniqueStrings(req.RoleIDs)
	if !req.GrantAllPermissions && len(roleIDs) == 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.repo.GrantAdmin(ctx, AdminGrantUpsert{UserID: req.UserID, GrantAllPermissions: req.GrantAllPermissions, RoleIDs: roleIDs}); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionAdminGrant,
		TargetType: AuditTargetTypeAdmin,
		TargetID:   req.UserID,
		TargetName: req.UserID,
		Details: AuditDetails{
			"type":                  string(AuditActionAdminGrant),
			"target_user_id":        req.UserID,
			"role_ids":              roleIDs,
			"grant_all_permissions": req.GrantAllPermissions,
		},
	}))
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
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "管理员不存在"}, nil
	}
	if target.IsSuperAdmin() && !actor.IsSuperAdmin() {
		return httpx.Forbidden(httpx.ReasonAdminProtectedTarget, "目标为平台管理员或超管，不能撤销"), nil
	}
	if err := s.repo.RevokeAdmin(ctx, userID); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	_ = s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionAdminRevoke,
		TargetType: AuditTargetTypeAdmin,
		TargetID:   userID,
		TargetName: userID,
		Details: AuditDetails{
			"type":           string(AuditActionAdminRevoke),
			"target_user_id": userID,
		},
	}))
	return nil, nil
}

func (s *Service) findRole(ctx context.Context, roleID string) (AdminRoleItem, bool, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return AdminRoleItem{}, false, err
	}
	for _, role := range roles {
		if role.RoleID == roleID {
			return role, true, nil
		}
	}
	return AdminRoleItem{}, false, nil
}

func diffStrings(current, previous []string) []string {
	set := make(map[string]struct{}, len(previous))
	for _, item := range previous {
		set[item] = struct{}{}
	}
	result := make([]string, 0)
	for _, item := range current {
		if _, ok := set[item]; !ok {
			result = append(result, item)
		}
	}
	return result
}
