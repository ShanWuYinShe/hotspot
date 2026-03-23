package admin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
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
	actor, apiErr, err := s.requirePermission(ctx, permissionRbacManage)
	if apiErr != nil || err != nil {
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
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}, nil
	}
	if err := s.repo.UpdateRole(ctx, req); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
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
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}, nil
	}
	if err := s.repo.DeleteRole(ctx, roleID); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
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
		return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.rbac.manage"}, nil
	}
	before, found, err := s.findRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}, nil
	}
	if err := s.repo.SetRolePermissions(ctx, req.RoleID, permissionKeys); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
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

func (s *Service) ListAuditLogs(ctx context.Context, req AuditLogListRequest) (PaginatedAuditLogs, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAuditRead)
	if apiErr != nil || err != nil {
		return PaginatedAuditLogs{}, apiErr, err
	}
	filter, apiErr := validateAuditFilter(req)
	if apiErr != nil {
		return PaginatedAuditLogs{}, apiErr, nil
	}
	page, err := s.repo.ListAuditLogs(ctx, filter, actor.HasPermission(permissionAuditReadIP))
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAuditLogs{}, apiErr, nil
		}
		return PaginatedAuditLogs{}, nil, err
	}
	resp := PaginatedAuditLogs{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAuditCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) GetAuditLog(ctx context.Context, logID string) (AuditLogItem, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAuditRead)
	if apiErr != nil || err != nil {
		return AuditLogItem{}, apiErr, err
	}
	if strings.TrimSpace(logID) == "" {
		return AuditLogItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetAuditLog(ctx, logID, actor.HasPermission(permissionAuditReadIP))
	if err != nil {
		return AuditLogItem{}, nil, err
	}
	if !found {
		return AuditLogItem{}, &httpx.APIError{Code: 10300, Message: "审计日志不存在"}, nil
	}
	return item, nil, nil
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

func (s *Service) UpdateUpload(ctx context.Context, req UpdateUploadConfigRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.Upload.ImageMaxSize != nil {
			if *req.Upload.ImageMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.ImageMaxSize = *req.Upload.ImageMaxSize
		}
		if req.Upload.VideoMaxSize != nil {
			if *req.Upload.VideoMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.VideoMaxSize = *req.Upload.VideoMaxSize
		}
		if req.Upload.AudioMaxSize != nil {
			if *req.Upload.AudioMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.AudioMaxSize = *req.Upload.AudioMaxSize
		}
		if req.Upload.FileMaxSize != nil {
			if *req.Upload.FileMaxSize <= 0 {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Upload.FileMaxSize = *req.Upload.FileMaxSize
		}
		if req.Upload.AllowedImageTypes != nil {
			cfg.Upload.AllowedImageTypes = sortAndUniqueStrings(req.Upload.AllowedImageTypes)
		}
		if req.Upload.AllowedVideoTypes != nil {
			cfg.Upload.AllowedVideoTypes = sortAndUniqueStrings(req.Upload.AllowedVideoTypes)
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "upload", before.Upload, updated.Upload)
}

func (s *Service) UpdateDefaults(ctx context.Context, req UpdateDefaultsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.DefaultLocation != nil {
			if req.DefaultLocation.Point != nil {
				cfg.Defaults.DefaultLocation.Point = *req.DefaultLocation.Point
			}
			if req.DefaultLocation.LocationName != nil {
				if strings.TrimSpace(*req.DefaultLocation.LocationName) == "" {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultLocation.LocationName = strings.TrimSpace(*req.DefaultLocation.LocationName)
			}
		}
		if req.DefaultGroupAvatar != nil {
			cfg.Defaults.DefaultGroupAvatar = strings.TrimSpace(*req.DefaultGroupAvatar)
		}
		if req.DefaultUserAvatar != nil {
			cfg.Defaults.DefaultUserAvatar = strings.TrimSpace(*req.DefaultUserAvatar)
		}
		if req.DefaultGroupSettings != nil {
			if req.DefaultGroupSettings.JoinMethod != nil {
				joinMethod := strings.TrimSpace(string(*req.DefaultGroupSettings.JoinMethod))
				if joinMethod == "" {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultGroupSettings.JoinMethod = *req.DefaultGroupSettings.JoinMethod
			}
			if req.DefaultGroupSettings.IsSearchable != nil {
				cfg.Defaults.DefaultGroupSettings.IsSearchable = *req.DefaultGroupSettings.IsSearchable
			}
			if req.DefaultGroupSettings.MaxMembers != nil {
				if *req.DefaultGroupSettings.MaxMembers <= 0 {
					return AppConfig{}, httpx.InvalidRequest("参数错误")
				}
				cfg.Defaults.DefaultGroupSettings.MaxMembers = *req.DefaultGroupSettings.MaxMembers
			}
			if req.DefaultGroupSettings.Location != nil {
				if req.DefaultGroupSettings.Location.Point != nil {
					cfg.Defaults.DefaultGroupSettings.Location.Point = *req.DefaultGroupSettings.Location.Point
				}
				if req.DefaultGroupSettings.Location.LocationName != nil {
					if strings.TrimSpace(*req.DefaultGroupSettings.Location.LocationName) == "" {
						return AppConfig{}, httpx.InvalidRequest("参数错误")
					}
					cfg.Defaults.DefaultGroupSettings.Location.LocationName = strings.TrimSpace(*req.DefaultGroupSettings.Location.LocationName)
				}
			}
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "defaults", before.Defaults, updated.Defaults)
}

func (s *Service) UpdateRateLimits(ctx context.Context, req UpdateRateLimitsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.RateLimits.GlobalPerHour, req.RateLimits.GlobalPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.UserPerHour, req.RateLimits.UserPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthRegisterPer10Minutes, req.RateLimits.AuthRegisterPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthLoginPer10Minutes, req.RateLimits.AuthLoginPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthGuestPer10Minutes, req.RateLimits.AuthGuestPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthRefreshPer10Minutes, req.RateLimits.AuthRefreshPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AuthResetPer10Minutes, req.RateLimits.AuthResetPer10Minutes); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.SearchPerMinute, req.RateLimits.SearchPerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.WritePerMinute, req.RateLimits.WritePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.MomentsCreatePerMinute, req.RateLimits.MomentsCreatePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.CommentsCreatePerMinute, req.RateLimits.CommentsCreatePerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.ReportsCreatePerHour, req.RateLimits.ReportsCreatePerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.UploadPresignPerHour, req.RateLimits.UploadPresignPerHour); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.WSSubscribePerSecond, req.RateLimits.WSSubscribePerSecond); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AdminReadPerMinute, req.RateLimits.AdminReadPerMinute); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.RateLimits.AdminWritePerMinute, req.RateLimits.AdminWritePerMinute); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "rate_limits", before.RateLimits, updated.RateLimits)
}

func (s *Service) UpdateResponseDefaults(ctx context.Context, req UpdateResponseDefaultsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.SearchSuggestCount, req.ResponseDefaults.SearchSuggestCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.TrendingTagsCount, req.ResponseDefaults.TrendingTagsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.HotMomentsCount, req.ResponseDefaults.HotMomentsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.ChatRecentMessagesCount, req.ResponseDefaults.ChatRecentMessagesCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.GroupAnnouncementsCount, req.ResponseDefaults.GroupAnnouncementsCount); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.ResponseDefaults.SystemAnnouncementsCount, req.ResponseDefaults.SystemAnnouncementsCount); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "response_defaults", before.ResponseDefaults, updated.ResponseDefaults)
}

func (s *Service) UpdateFeatures(ctx context.Context, req UpdateFeaturesRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.EnablePublicRegistration != nil {
			cfg.Features.EnablePublicRegistration = *req.EnablePublicRegistration
		}
		if req.EnableGuestLogin != nil {
			cfg.Features.EnableGuestLogin = *req.EnableGuestLogin
		}
		if req.EnableMoments != nil {
			cfg.Features.EnableMoments = *req.EnableMoments
		}
		if req.EnableNearbyPeople != nil {
			cfg.Features.EnableNearbyPeople = *req.EnableNearbyPeople
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "features", before.Features, updated.Features)
}

func (s *Service) UpdateLimits(ctx context.Context, req UpdateLimitsRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if err := applyPositiveIntPatch(&cfg.Limits.MaxGroupMembers, req.Limits.MaxGroupMembers); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MaxMomentTags, req.Limits.MaxMomentTags); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MaxMomentImages, req.Limits.MaxMomentImages); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.MomentDescriptionMaxLength, req.Limits.MomentDescriptionMaxLength); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.CommentMaxLength, req.Limits.CommentMaxLength); err != nil {
			return AppConfig{}, err
		}
		if err := applyPositiveIntPatch(&cfg.Limits.SearchRadiusMax, req.Limits.SearchRadiusMax); err != nil {
			return AppConfig{}, err
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "limits", before.Limits, updated.Limits)
}

func (s *Service) UpdateContact(ctx context.Context, req UpdateContactRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAppConfigWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	before, err := s.provider.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	updated, apiErr, err := s.provider.UpdateAppConfig(ctx, actor.UserID, func(cfg AppConfig) (AppConfig, *httpx.APIError) {
		if req.SupportEmail != nil {
			if !strings.Contains(strings.TrimSpace(*req.SupportEmail), "@") {
				return AppConfig{}, httpx.InvalidRequest("参数错误")
			}
			cfg.Contact.SupportEmail = strings.TrimSpace(*req.SupportEmail)
		}
		if req.PrivacyPolicyURL != nil {
			cfg.Contact.PrivacyPolicyURL = strings.TrimSpace(*req.PrivacyPolicyURL)
		}
		if req.TermsOfServiceURL != nil {
			cfg.Contact.TermsOfServiceURL = strings.TrimSpace(*req.TermsOfServiceURL)
		}
		if req.OfficialWebsite != nil {
			cfg.Contact.OfficialWebsite = strings.TrimSpace(*req.OfficialWebsite)
		}
		return cfg, nil
	})
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return nil, s.appendConfigAudit(ctx, actor, "contact", before.Contact, updated.Contact)
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

func validateAuditFilter(req AuditLogListRequest) (AuditLogFilter, *httpx.APIError) {
	filter := AuditLogFilter{Action: req.Action, OperatorID: strings.TrimSpace(req.OperatorID), TargetType: req.TargetType, TargetID: strings.TrimSpace(req.TargetID)}
	if req.Pagination != nil {
		filter.Pagination = *req.Pagination
	}
	if strings.TrimSpace(req.StartTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartTime))
		if err != nil {
			return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
		}
		filter.StartTime = &parsed
	}
	if strings.TrimSpace(req.EndTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndTime))
		if err != nil {
			return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
		}
		filter.EndTime = &parsed
	}
	if filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime) {
		return AuditLogFilter{}, httpx.InvalidRequest("参数错误")
	}
	if _, apiErr := decodeAuditCursor(filter.Pagination.Cursor); apiErr != nil {
		return AuditLogFilter{}, apiErr
	}
	return filter, nil
}

func applyPositiveIntPatch(target *int, patch *int) *httpx.APIError {
	if patch == nil {
		return nil
	}
	if *patch <= 0 {
		return httpx.InvalidRequest("参数错误")
	}
	*target = *patch
	return nil
}

func newAuditEntry(ctx context.Context, actor AdminActor, entry AuditLogEntry) AuditLogEntry {
	if entry.LogID == "" {
		entry.LogID = "audit_" + observability.RequestIDFromContext(ctx)
	}
	if entry.Operator.UserID == "" {
		entry.Operator = UserInfo{UserID: actor.UserID, Nickname: actor.UserID}
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if entry.IPAddress == "" {
		entry.IPAddress = requestIPFromContext(ctx)
	}
	if entry.DeviceSummary == "" {
		entry.DeviceSummary = deviceSummaryFromContext(ctx)
	}
	if entry.IPLocation == "" && entry.IPAddress != "" {
		entry.IPLocation = "unknown"
	}
	return entry
}

func requestIPFromContext(ctx context.Context) string {
	req, _ := requestFromContext(ctx)
	if req == nil {
		return ""
	}
	if ip := strings.TrimSpace(req.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if ip := strings.TrimSpace(req.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if forwarded := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(req.RemoteAddr)
}

func deviceSummaryFromContext(ctx context.Context) string {
	req, _ := requestFromContext(ctx)
	if req == nil {
		return ""
	}
	ua := strings.TrimSpace(req.Header.Get("User-Agent"))
	if ua == "" {
		return ""
	}
	if len(ua) > 120 {
		return ua[:120]
	}
	return ua
}

type requestContextKey struct{}

func withRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, req)
}

func requestFromContext(ctx context.Context) (*http.Request, bool) {
	req, ok := ctx.Value(requestContextKey{}).(*http.Request)
	return req, ok
}

func (s *Service) appendConfigAudit(ctx context.Context, actor AdminActor, configKey string, oldValue, newValue any) error {
	return s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, AuditLogEntry{
		Action:     AuditActionConfigUpdate,
		TargetType: AuditTargetTypeConfig,
		TargetID:   configKey,
		TargetName: configKey,
		Details: AuditDetails{
			"type":       string(AuditActionConfigUpdate),
			"config_key": configKey,
			"old_value":  oldValue,
			"new_value":  newValue,
		},
	}))
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
