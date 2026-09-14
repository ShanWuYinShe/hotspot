package admin

import (
	"context"
	"hotspot/backend-go/internal/platform/httpx"
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
		if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.unban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("user"), TargetID: req.UserID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.unban", "reason": strings.TrimSpace(req.Reason), "no_op": true}}); err != nil {
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
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.unban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("user"), TargetID: req.UserID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.unban", "reason": strings.TrimSpace(req.Reason)}}); err != nil {
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
	item, found, err := s.repo.DismissGroup(ctx, req.GroupID)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "群组不存在"}, nil
	}
	event := realtime.Event{Type: realtime.EventTypeGroupDismissed, Payload: map[string]any{"group_id": req.GroupID, "dismissed_at": time.Now().UTC().Format(time.RFC3339), "reason": "policy_violation"}}
	// 仅按群广播一次：逐成员 BroadcastToUser 会让在线成员各收到两份重复事件；
	// 离线成员由 resume 兜底（解散为持久状态，重连后群已不可见）。
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("group.dismiss"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("group"), TargetID: req.GroupID, TargetName: item.Group.Name, Details: map[string]any{"type": "group.dismiss", "reason": strings.TrimSpace(req.Reason)}}); err != nil {
		return nil, err
	}
	return nil, nil
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
