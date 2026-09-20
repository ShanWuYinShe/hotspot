package admin

import (
	"context"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"strings"
	"time"
)

func (s *Service) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (PaginatedAdminUsers, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionUsersRead)
	if apiErr != nil || err != nil {
		return PaginatedAdminUsers{}, apiErr, err
	}
	if err := validateAdminUserSearchRequest(req); err != nil {
		return PaginatedAdminUsers{}, err, nil
	}
	page, err := s.repo.SearchUsers(ctx, req)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return PaginatedAdminUsers{}, apiErr, nil
		}
		return PaginatedAdminUsers{}, nil, err
	}
	// IP 字段为敏感 PII（spec api_spec_admin.md admin.users.read_ip）：权限不足时字段为 null
	if !actor.HasPermission(permissionUsersReadIP) {
		for i := range page.Items {
			page.Items[i].RegisterIP = nil
			page.Items[i].LastLoginIP = nil
		}
	}
	resp := PaginatedAdminUsers{Items: page.Items}
	if page.Next != nil {
		c := encodeAdminUserCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

func (s *Service) OpenAdminUser(ctx context.Context, userID string) (AdminUserItem, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionUsersRead)
	if apiErr != nil || err != nil {
		return AdminUserItem{}, apiErr, err
	}
	if strings.TrimSpace(userID) == "" {
		return AdminUserItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return AdminUserItem{}, apiErr, nil
		}
		return AdminUserItem{}, nil, err
	}
	if !found {
		return AdminUserItem{}, &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}, nil
	}
	// IP 字段为敏感 PII（spec api_spec_admin.md admin.users.read_ip）：权限不足时字段为 null
	if !actor.HasPermission(permissionUsersReadIP) {
		item.RegisterIP = nil
		item.LastLoginIP = nil
	}
	return item, nil, nil
}

func (s *Service) SetUserBan(ctx context.Context, req AdminSetUserBanRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionUsersWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if req.Active {
		return s.banUser(ctx, actor, req.UserID, req.Reason, req.DurationDays, true)
	}
	if strings.TrimSpace(req.UserID) == "" || len(req.Reason) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}, nil
	}
	state, ok, err := s.repo.GetUserBanState(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	// 解封未封禁用户视为 no-op 成功（契约：设状态操作重复达同状态返回成功，
	// 见 api_spec_errors.md）；spec api_spec_admin.md users/set_ban 安全约束：
	// 封禁/解封操作均记录审计日志（含 no-op 解封尝试，便于审计追溯）。
	if !ok || state.Status != UserAccountStatusBanned {
		if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.unban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetType("user"), TargetID: req.UserID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.unban", "reason": strings.TrimSpace(req.Reason), "no_op": true}}); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if _, isAdmin, err := s.repo.GetActor(ctx, req.UserID); err != nil {
		return nil, err
	} else if isAdmin && !actor.IsSuperAdmin() {
		return httpx.Forbidden(httpx.ReasonAdminProtectedTarget, "目标为平台管理员，不能解封"), nil
	}
	if err := s.repo.SetUserBanState(ctx, req.UserID, UserBanState{Status: UserAccountStatusNormal}); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	s.invalidateBanCache(req.UserID)
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.unban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetType("user"), TargetID: req.UserID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.unban", "reason": strings.TrimSpace(req.Reason)}}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) SearchGroups(ctx context.Context, req AdminGroupSearchRequest) (PaginatedAdminGroups, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionGroupsRead); apiErr != nil || err != nil {
		return PaginatedAdminGroups{}, apiErr, err
	}
	if req.Pagination != nil {
		if _, apiErr := decodeAdminGroupCursor(req.Pagination.Cursor); apiErr != nil {
			return PaginatedAdminGroups{}, apiErr, nil
		}
	}
	page, err := s.repo.SearchGroups(ctx, req)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return PaginatedAdminGroups{}, apiErr, nil
		}
		return PaginatedAdminGroups{}, nil, err
	}
	resp := PaginatedAdminGroups{Items: page.Items}
	if page.Next != nil {
		c := encodeAdminGroupCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

func (s *Service) OpenAdminGroup(ctx context.Context, groupID string) (AdminGroupItem, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionGroupsRead); apiErr != nil || err != nil {
		return AdminGroupItem{}, apiErr, err
	}
	if strings.TrimSpace(groupID) == "" {
		return AdminGroupItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return AdminGroupItem{}, nil, err
	}
	if !found {
		return AdminGroupItem{}, &httpx.APIError{Code: httpx.CodeNotFound, Message: "群组不存在"}, nil
	}
	return item, nil, nil
}

func (s *Service) DismissGroup(ctx context.Context, req AdminDismissGroupRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionGroupsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if strings.TrimSpace(req.GroupID) == "" || len(req.Reason) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	_, apiErr, err = s.dismissGroupAndTrace(ctx, actor, req.GroupID, req.Reason)
	return apiErr, err
}

// dismissGroupAndTrace 执行群解散并补齐两级契约动作：成员 GroupDismissed 广播 +
// group.dismiss 审计。admin 端点（groups/dismiss）与举报裁决（reports/judge
// resolve 群目标）共用，保证两条解散路径语义一致（spec api_spec_admin.md
// groups/dismiss 安全约束）。
// 审计写入失败按 best-effort 处理（记错误日志，不回滚已生效的解散）：解散不可逆，
// 失败报错只会诱导管理员重试并撞上「群组不存在」。
func (s *Service) dismissGroupAndTrace(ctx context.Context, actor AdminActor, groupID, reason string) (AdminGroupItem, *httpx.APIError, error) {
	item, found, err := s.repo.DismissGroup(ctx, groupID)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return AdminGroupItem{}, apiErr, nil
		}
		return AdminGroupItem{}, nil, err
	}
	if !found {
		return AdminGroupItem{}, &httpx.APIError{Code: httpx.CodeNotFound, Message: "群组不存在"}, nil
	}
	event := realtime.Event{Type: realtime.EventTypeGroupDismissed, Payload: groups.GroupDismissedPayload{GroupID: groupID, DismissedAt: time.Now().UTC().Format(time.RFC3339), Reason: "policy_violation"}}
	// 仅按群广播一次：逐成员 BroadcastToUser 会让在线成员各收到两份重复事件；
	// 离线成员由 resume 兜底（解散为持久状态，重连后群已不可见）。
	realtime.DefaultHub().BroadcastToGroup(groupID, event)
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("group.dismiss"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetType("group"), TargetID: groupID, TargetName: item.Group.Name, Details: map[string]any{"type": "group.dismiss", "reason": strings.TrimSpace(reason)}}); err != nil {
		observability.L().Error("admin: group dismiss audit log append failed", "error", err, "group_id", groupID)
	}
	return item, nil, nil
}

func validateAdminUserSearchRequest(req AdminUserSearchRequest) *httpx.APIError {
	if req.Pagination != nil {
		if _, apiErr := decodeAdminUserCursor(req.Pagination.Cursor); apiErr != nil {
			return apiErr
		}
	}
	if req.Status != "" && req.Status != UserAccountStatusNormal && req.Status != UserAccountStatusBanned {
		return httpx.InvalidRequest("参数错误")
	}
	return nil
}
