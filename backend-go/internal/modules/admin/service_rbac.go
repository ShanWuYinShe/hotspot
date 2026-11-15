package admin

import (
	"context"
	"errors"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

func (s *Service) GetMe(ctx context.Context) (AdminPermissions, *httpx.APIError, error) {
	op := observability.Begin("admin.get_me")
	actor, found, err := s.actorFromContext(ctx)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			op.Warn("auth failed", "error", apiErr)
			return AdminPermissions{}, apiErr, nil
		}
		op.Fail(err)
		return AdminPermissions{}, nil, err
	}
	if !found {
		op.Warn("not admin")
		return AdminPermissions{}, httpx.Forbidden(httpx.ReasonNotAdmin, "非平台管理员"), nil
	}
	op.Complete("user_id", actor.UserID)
	return actor.Response(), nil, nil
}

func (s *Service) ListPermissions(ctx context.Context) (AdminPermissionDictionary, *httpx.APIError, error) {
	op := observability.Begin("admin.permissions.fetch")
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		op.Warn("permission denied")
		return AdminPermissionDictionary{}, apiErr, err
	}
	items, err := s.repo.ListPermissions(ctx)
	if err != nil {
		op.Fail(err, "stage", "query")
		return AdminPermissionDictionary{}, nil, err
	}
	op.Complete("count", len(items))
	return AdminPermissionDictionary{Items: items}, nil, nil
}

func (s *Service) ListRoles(ctx context.Context) (AdminRoleList, *httpx.APIError, error) {
	op := observability.Begin("admin.list_roles")
	if _, apiErr, err := s.requirePermission(ctx, permissionRbacManage); apiErr != nil || err != nil {
		op.Warn("permission denied")
		return AdminRoleList{}, apiErr, err
	}
	items, err := s.repo.ListRoles(ctx)
	if err != nil {
		op.Fail(err, "stage", "query")
		return AdminRoleList{}, nil, err
	}
	op.Complete("count", len(items))
	return AdminRoleList{Items: items}, nil, nil
}

// CreateRole 创建角色并写 role.create 审计（spec api_spec_admin.md 审计动作
// 清单要求 role.* 可追溯）：角色落库与审计同一事务，审计写失败整体回滚，
// 管理员可安全重试；此前审计失败仅记日志，会出现「角色已创建但无审计」。
// 未装配 TxRunner（单测/内存装配）时退化为直跑，该回退路径无法回滚。
func (s *Service) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, *httpx.APIError, error) {
	op := observability.Begin("admin.create_role", "role_name", req.Name)
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		op.Warn("permission denied")
		return AdminRoleItem{}, apiErr, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return AdminRoleItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	var item AdminRoleItem
	apiErr, err = s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		created, createErr := s.repo.CreateRole(txCtx, req)
		if createErr != nil {
			if apiErr := asAPIError(createErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, createErr
		}
		item = created
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
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
	})
	if apiErr != nil {
		op.Warn("business error", "error", apiErr)
		return AdminRoleItem{}, apiErr, nil
	}
	if err != nil {
		op.Fail(err)
		return AdminRoleItem{}, nil, err
	}
	return item, nil, nil
}

// UpdateRole 更新角色名称/描述并写 role.update 审计：读前态、更新、读后态与
// 审计同一事务（审计快照与库内实际变更一致），审计写失败整体回滚（角色未变），
// 管理员可安全重试；此前审计失败仅记日志。未装配 TxRunner 时退化为直跑。
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
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		before, found, findErr := s.findRole(txCtx, req.RoleID)
		if findErr != nil {
			return nil, findErr
		}
		if !found {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
		}
		if updateErr := s.repo.UpdateRole(txCtx, req); updateErr != nil {
			if apiErr := asAPIError(updateErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, updateErr
		}
		after, found, findErr := s.findRole(txCtx, req.RoleID)
		if findErr != nil {
			return nil, findErr
		}
		if !found {
			// 同事务内刚更新成功却查不到（数据异常）：宁可回滚也不写与库内状态
			// 不一致的审计。
			return nil, errors.New("admin: role disappeared after update")
		}
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
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
	})
}

// DeleteRole 删除（软删）角色并写 role.delete 审计：读前态、删除与审计同一
// 事务，审计写失败整体回滚（角色未删），管理员可安全重试；此前审计失败仅记
// 日志。未装配 TxRunner 时退化为直跑。
func (s *Service) DeleteRole(ctx context.Context, roleID string) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if roleID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		before, found, findErr := s.findRole(txCtx, roleID)
		if findErr != nil {
			return nil, findErr
		}
		if !found {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
		}
		if deleteErr := s.repo.DeleteRole(txCtx, roleID); deleteErr != nil {
			if apiErr := asAPIError(deleteErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, deleteErr
		}
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
			Action:     AuditActionRoleDelete,
			TargetType: AuditTargetTypeRole,
			TargetID:   before.RoleID,
			TargetName: before.Name,
			Details: AuditDetails{
				"type":      string(AuditActionRoleDelete),
				"role_name": before.Name,
			},
		}))
	})
}

// SetRolePermissions 替换角色权限并写 role.set_permissions 审计：读前态、权限
// 替换与审计同一事务，审计写失败整体回滚（权限未变），管理员可安全重试；此前
// 审计失败仅记日志。未装配 TxRunner 时退化为直跑。
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
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		before, found, findErr := s.findRole(txCtx, req.RoleID)
		if findErr != nil {
			return nil, findErr
		}
		if !found {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}, nil
		}
		if setErr := s.repo.SetRolePermissions(txCtx, req.RoleID, permissionKeys); setErr != nil {
			if apiErr := asAPIError(setErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, setErr
		}
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
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
	})
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
		c := encodeAdminCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

// GrantAdmin 授予平台管理员并写 admin.award 审计：授权落库与审计同一事务，
// 审计写失败整体回滚（授权未生效），管理员可安全重试；此前审计失败仅记日志，
// 会出现「已授权但无审计」。未装配 TxRunner 时退化为直跑。
func (s *Service) GrantAdmin(ctx context.Context, req AdminRbacGrantAdminRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.UserID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.AwardAllPermissions && len(req.RoleIDs) > 0 {
		return httpx.InvalidRequest("award_all_permissions 与 role_ids 互斥"), nil
	}
	if req.AwardAllPermissions && !actor.IsSuperAdmin() {
		return httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permissionRbacManage), nil
	}
	roleIDs := sortAndUniqueStrings(req.RoleIDs)
	if !req.AwardAllPermissions && len(roleIDs) == 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		if grantErr := s.repo.GrantAdmin(txCtx, AdminGrantUpsert{UserID: req.UserID, GrantAllPermissions: req.AwardAllPermissions, RoleIDs: roleIDs}); grantErr != nil {
			if apiErr := asAPIError(grantErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, grantErr
		}
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
			Action:     AuditActionAdminAward,
			TargetType: AuditTargetTypeAdmin,
			TargetID:   req.UserID,
			TargetName: req.UserID,
			Details: AuditDetails{
				"type":                  string(AuditActionAdminAward),
				"target_user_id":        req.UserID,
				"role_ids":              roleIDs,
				"award_all_permissions": req.AwardAllPermissions,
			},
		}))
	})
}

// RevokeAdmin 撤销平台管理员并写 admin.revoke 审计：撤销与审计同一事务，审计
// 写失败整体回滚（管理员未被撤销），管理员可安全重试；此前审计失败仅记日志。
// 未装配 TxRunner 时退化为直跑。
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
	// 自我保护（与 SetUserBan 的防御一致）：超管撤销自己后若无其他超管，
	// 后台永久失联，只能靠 DB 手工修数据。
	if actor.UserID == userID {
		return httpx.Forbidden(httpx.ReasonAdminProtectedTarget, "不能撤销自己的管理员权限"), nil
	}
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		if revokeErr := s.repo.RevokeAdmin(txCtx, userID); revokeErr != nil {
			if apiErr := asAPIError(revokeErr); apiErr != nil {
				return apiErr, nil
			}
			return nil, revokeErr
		}
		return nil, s.repo.AppendAuditLog(txCtx, newAuditEntry(txCtx, actor, AuditLogEntry{
			Action:     AuditActionAdminRevoke,
			TargetType: AuditTargetTypeAdmin,
			TargetID:   userID,
			TargetName: userID,
			Details: AuditDetails{
				"type":           string(AuditActionAdminRevoke),
				"target_user_id": userID,
			},
		}))
	})
}

// findRole 按 role_id 直查（审查 P2-23）：此前 ListRoles 全量扫描 + 线性
// 查找，角色数增长后每次角色写操作的审计快照都是 O(全表)。
func (s *Service) findRole(ctx context.Context, roleID string) (AdminRoleItem, bool, error) {
	return s.repo.GetRole(ctx, roleID)
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
