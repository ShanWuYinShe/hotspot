package groups

import (
	"context"
	"errors"

	"hotspot/backend-go/internal/platform/httpx"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/rbac"
	platformtypes "hotspot/backend-go/internal/platform/types"
)

// RoleOf 实现 rbac.GroupRoleResolver：解析当前调用者在目标群的角色（守卫子树用）。
// 未登录/非成员返回 (GroupRole(""), false, nil)。
func (s *Service) RoleOf(ctx context.Context, groupID string) (rbac.GroupRole, bool, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return "", false, nil
	}
	item, found, err := s.repo.Get(ctx, groupID)
	if err != nil || !found {
		return "", false, err
	}
	if item.ViewerState == nil {
		return "", false, nil
	}
	return rbac.GroupRole(item.ViewerState.Role), true, nil
}

// GetSettings 返回群管理视角设置（groups/admin/get_settings；含 has_password/
// is_searchable/members_searchable/max_members 等管理字段，spec api_spec_types.md GroupSettings）。
func (s *Service) GetSettings(ctx context.Context, groupID string) (GroupSettings, bool, error) {
	item, ok, err := s.repo.Get(ctx, groupID)
	if err != nil || !ok {
		return GroupSettings{}, ok, err
	}
	return GroupSettings{
		JoinMethod:        item.JoinMethod,
		RequiresApproval:  item.RequiresApproval,
		HasPassword:       item.HasPassword,
		IsSearchable:      item.IsSearchable,
		MembersSearchable: item.MembersSearchable,
		MaxMembers:        item.MaxMembers,
		Location: Location{
			Point:        item.Location.Point,
			LocationName: item.Location.LocationName,
		},
	}, true, nil
}

// ListAdminMembers 返回管理视角成员表（muted_until 真值，null=未禁言）。
func (s *Service) ListAdminMembers(ctx context.Context, req ListAdminGroupMembersRequest) ([]AdminGroupMemberItem, string, error) {
	limit := platformtypes.PaginationLimitWith(req.Pagination, 20, 100)
	return s.repo.ListAdminMembers(ctx, req.GroupID, req.Role, limit, platformtypes.PaginationCursor(req.Pagination))
}

// RecallGroupMessageAsAdmin 经守卫子树撤回任意成员消息（无时间窗）。
// 实际撤回能力由 bootstrap 注入的 RecallMessageAsAdmin（groupmessages 实现）承担。
func (s *Service) RecallGroupMessageAsAdmin(ctx context.Context, req RecallGroupMessageAdminRequest) error {
	if RecallMessageAsAdmin == nil {
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
	}
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "Access Token 无效或过期"}
	}
	if err := RecallMessageAsAdmin(ctx, req.GroupID, req.MessageID, principal.UserID); err != nil {
		var apiErr *httpx.APIError
		if errors.As(err, &apiErr) {
			return err
		}
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
	}
	return nil
}
