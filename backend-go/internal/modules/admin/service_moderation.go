package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

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
		resp.Items = append(resp.Items, AdminTagItem{TagID: "tag_" + item.Name, Name: item.Name, UsageCount: item.UsageCount, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	if page.Next != nil {
		c := encodeTagCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

// SetTagBan 设置标签屏蔽状态（两态统一端点，spec api_spec_admin.md
// admin/tags/set_ban）。active=true 软屏蔽（可选 hard_delete 物理删除历史
// 引用）；active=false 解屏蔽（对未屏蔽标签 no-op 成功，恢复同名标签可
// 创建/命中）。屏蔽/解屏蔽均落审计（no-op 解屏蔽也落，与 users/set_ban
// 的审计写法对齐）。
func (s *Service) SetTagBan(ctx context.Context, req AdminSetTagBanRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionTagsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	tagName := normalizeTagName(req.Name)
	if tagName == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	// hard_delete 仅 active=true 有效：解屏蔽属恢复操作，与不可逆物理删除
	// 语义冲突，同传视为参数冲突 10000。
	if !req.Active && req.HardDelete {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if err := s.moderationStore.SetTagBan(ctx, tagName, req.Active, req.HardDelete); err != nil {
		if errors.Is(err, errTagNotFound) {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "标签不存在"}, nil
		}
		return nil, err
	}
	details := AuditDetails{"type": string(AuditActionTagSetBan), "name": tagName, "active": req.Active}
	if req.HardDelete {
		details["hard_delete"] = true
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionTagSetBan, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeTag, TargetID: tagName, TargetName: tagName, Details: details}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) ListAdminAnnouncements(ctx context.Context, req ListAdminAnnouncementsRequest) (PaginatedAdminAnnouncements, *httpx.APIError, error) {
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
		c := encodeAnnouncementCursor(announcementCursor{CreatedAt: page.Next.CreatedAt, AnnouncementID: page.Next.AnnouncementID})
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

// CreateAnnouncement 发布公告（spec api_spec_admin.md announcements/create：
// Response 为 ApiResponse<AdminAnnouncementItem>，返回完整公告对象供客户端
// 直接渲染）。
func (s *Service) CreateAnnouncement(ctx context.Context, req CreateAnnouncementRequest) (AdminAnnouncementItem, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite)
	if apiErr != nil || err != nil {
		return AdminAnnouncementItem{}, apiErr, err
	}
	if apiErr := validateAnnouncementCreate(req); apiErr != nil {
		return AdminAnnouncementItem{}, apiErr, nil
	}
	title := strings.TrimSpace(req.Title)
	input := systemmodule.AnnouncementCreateInput{
		Actor: systemmodule.AnnouncementActor{UserID: actor.UserID, Nickname: actor.UserID},
		Title: title, Content: strings.TrimSpace(req.Content), Severity: req.Severity,
		ActionURL: strings.TrimSpace(req.ActionURL), SendPush: req.SendPush,
	}
	item, err := s.moderationStore.CreateAnnouncement(ctx, input)
	if err != nil {
		return AdminAnnouncementItem{}, nil, err
	}
	if s.pushService != nil {
		// 全量 fanout 按 user_id 升序 keyset 分批：此前复用 SearchUsers
		// （limit 被 NormalizeLimit 钳到 100），公告只落库给前 100 个用户。
		const fanoutBatch = 500
		afterUserID := ""
		var lastNotificationID string
		for {
			batch, listErr := s.repo.ListUserIDsAfter(ctx, afterUserID, fanoutBatch)
			if listErr != nil {
				return AdminAnnouncementItem{}, nil, listErr
			}
			if len(batch) == 0 {
				break
			}
			notificationID, err := s.pushService.AppendSystemAnnouncement(ctx, batch, item.PublicItem())
			if err != nil {
				return AdminAnnouncementItem{}, nil, err
			}
			lastNotificationID = notificationID
			if len(batch) < fanoutBatch {
				break
			}
			afterUserID = batch[len(batch)-1]
		}
		// BroadcastToAll 是一次全体在线广播，必须移出批次循环（否则广播 N 次）。
		// 跨批次落库 notify_id 不同（批内 row_number 后缀），客户端去重以
		// payload 里的 announcement_id 为准，notify_id 仅作展示。
		if req.SendPush && lastNotificationID != "" {
			s.pushService.PushSystemAnnouncement(item.PublicItem(), lastNotificationID)
		}
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementCreate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: item.AnnouncementID, TargetName: title, Details: AuditDetails{"type": string(AuditActionAnnouncementCreate), "title": title}}); err != nil {
		return AdminAnnouncementItem{}, nil, err
	}
	return toAdminAnnouncementItem(item), nil, nil
}

// UpdateAnnouncement 更新公告（spec api_spec_admin.md announcements/update：
// 内容类更新返回更新后完整对象 ApiResponse<AdminAnnouncementItem>）。
func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateAnnouncementRequest) (AdminAnnouncementItem, *httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite)
	if apiErr != nil || err != nil {
		return AdminAnnouncementItem{}, apiErr, err
	}
	if apiErr := validateAnnouncementUpdate(req); apiErr != nil {
		return AdminAnnouncementItem{}, apiErr, nil
	}
	current, found, err := s.moderationStore.GetAnnouncement(ctx, req.AnnouncementID)
	if err != nil {
		return AdminAnnouncementItem{}, nil, err
	}
	if !found || current.IsDeleted() {
		return AdminAnnouncementItem{}, &httpx.APIError{Code: httpx.CodeNotFound, Message: "公告不存在"}, nil
	}
	changedFields := make([]string, 0, 4)
	if req.Title != nil && strings.TrimSpace(*req.Title) != current.Title {
		changedFields = append(changedFields, "title")
	}
	if req.Content != nil && strings.TrimSpace(*req.Content) != current.Content {
		changedFields = append(changedFields, "content")
	}
	if req.Severity != nil && *req.Severity != current.Severity {
		changedFields = append(changedFields, "severity")
	}
	currentActionURL := strings.TrimSpace(current.ActionURL)
	if req.ActionURL != nil && strings.TrimSpace(*req.ActionURL) != currentActionURL {
		changedFields = append(changedFields, "action_url")
	}
	updatedTitle := current.Title
	if req.Title != nil {
		updatedTitle = strings.TrimSpace(*req.Title)
	}
	updated, err := s.moderationStore.UpdateAnnouncement(ctx, systemmodule.AnnouncementUpdateInput{
		AnnouncementID: req.AnnouncementID,
		Title:          trimOptional(req.Title),
		Content:        trimOptional(req.Content),
		Severity:       req.Severity,
		ActionURL:      trimOptional(req.ActionURL),
	})
	if err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) {
			return AdminAnnouncementItem{}, &httpx.APIError{Code: httpx.CodeNotFound, Message: "公告不存在"}, nil
		}
		return AdminAnnouncementItem{}, nil, err
	}
	details := AuditDetails{"type": string(AuditActionAnnouncementUpdate), "title": updatedTitle}
	if len(changedFields) > 0 {
		details["changed_fields"] = changedFields
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementUpdate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: req.AnnouncementID, TargetName: updatedTitle, Details: details}); err != nil {
		return AdminAnnouncementItem{}, nil, err
	}
	return toAdminAnnouncementItem(updated), nil, nil
}

func (s *Service) DeleteAnnouncement(ctx context.Context, announcementID string) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	current, found, err := s.moderationStore.GetAnnouncement(ctx, announcementID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "公告不存在"}, nil
	}
	if current.IsDeleted() {
		return nil, nil
	}
	if err := s.moderationStore.DeleteAnnouncement(ctx, announcementID); err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "公告不存在"}, nil
		}
		return nil, err
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementDelete, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: announcementID, TargetName: current.Title, Details: AuditDetails{"type": string(AuditActionAnnouncementDelete), "title": current.Title}}); err != nil {
		return nil, err
	}
	return nil, nil
}

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
	memberIDs, err := s.repo.ListGroupMemberIDs(ctx, req.GroupID)
	if err != nil {
		return nil, err
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
	for _, userID := range memberIDs {
		realtime.DefaultHub().BroadcastToUser(userID, event)
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, event)
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("group.dismiss"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("group"), TargetID: req.GroupID, TargetName: item.Group.Name, Details: map[string]any{"type": "group.dismiss", "reason": strings.TrimSpace(req.Reason)}}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) SearchReports(ctx context.Context, req AdminReportListRequest) (PaginatedAdminReports, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionReportsRead); apiErr != nil || err != nil {
		return PaginatedAdminReports{}, apiErr, err
	}
	if req.Pagination != nil {
		if _, apiErr := decodeAdminReportCursor(req.Pagination.Cursor); apiErr != nil {
			return PaginatedAdminReports{}, apiErr, nil
		}
	}
	page, err := s.repo.SearchReports(ctx, req)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return PaginatedAdminReports{}, apiErr, nil
		}
		return PaginatedAdminReports{}, nil, err
	}
	resp := PaginatedAdminReports{Items: page.Items}
	if page.Next != nil {
		c := encodeAdminReportCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

func (s *Service) HandleReport(ctx context.Context, req JudgeAdminReportRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionReportsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	note := strings.TrimSpace(req.Note)
	if strings.TrimSpace(req.ReportID) == "" || !isValidReportAction(req.Action) || len(note) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.BanDurationDays != nil && *req.BanDurationDays < 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	record, found, err := s.repo.GetReportHandleRecord(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "举报不存在"}, nil
	}
	if record.Item.Status != ReportStatusPending {
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "举报已处理"}, nil
	}
	// 权限蕴含（spec admin_moderation.md → reports/judge）：resolve 且携带
	// ban_duration_days 时实际执行封禁，须同时具备 admin.users.write，
	// 防止低权限管理员经举报通道绕过用户封禁权限边界。
	if req.Action == ReportActionResolve && req.BanDurationDays != nil && !actor.HasPermission(permissionUsersWrite) {
		return httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permissionUsersWrite), nil
	}
	if req.Action == ReportActionResolve {
		switch record.Item.TargetType {
		case ReportTargetTypeGroup:
			if _, _, err := s.repo.DismissGroup(ctx, record.Item.TargetID); err != nil {
				if apiErr := asAPIError(err); apiErr != nil {
					return apiErr, nil
				}
				return nil, err
			}
		case ReportTargetTypeUser:
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeMoment:
			if err := s.repo.DeleteMoment(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeComment:
			if err := s.repo.DeleteComment(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeMessage:
			if err := s.repo.RecallGroupMessage(ctx, record.Item.TargetID); err != nil {
				if err := s.repo.RecallPrivateMessage(ctx, record.Item.TargetID); err != nil {
					return nil, err
				}
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		}
	}
	_, ok, err := s.repo.HandleReport(ctx, req.ReportID, req.Action, note, &UserInfo{UserID: actor.UserID, Nickname: actor.UserID})
	if err != nil {
		if errors.Is(err, errReportAlreadyHandled) {
			// 并发裁决败者：副作用（解散/封禁/删内容）可能已由获胜方执行，
			// 直接返回已处理冲突，不再重复执行也不覆盖裁决结果。
			return &httpx.APIError{Code: httpx.CodeConflict, Message: "举报已处理"}, nil
		}
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	if !ok {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "举报不存在"}, nil
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("report.judge"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("report"), TargetID: req.ReportID, TargetName: req.ReportID, Details: map[string]any{"type": "report.judge", "report_id": req.ReportID, "action": string(req.Action), "note": note}}); err != nil {
		return nil, err
	}
	// Notify the reporter about the outcome (resolve/dismiss only; escalate
	// keeps the report pending and does not notify).
	if req.Action == ReportActionResolve || req.Action == ReportActionDismiss {
		s.notifyReportHandled(ctx, record.Item.Reporter.UserID, req.ReportID, string(record.Item.TargetType), record.Item.TargetID, note, req.Action == ReportActionResolve)
	}
	return nil, nil
}

// banReportedUser 按举报处理附带的天数封禁目标用户（nil = 不封禁，跳过）。
func (s *Service) banReportedUser(ctx context.Context, actor AdminActor, targetUserID, note string, banDays *int) (*httpx.APIError, error) {
	if banDays == nil {
		return nil, nil
	}
	return s.banUser(ctx, actor, targetUserID, note, banDays, false)
}

// notifyReportHandled persists the report_resolved/report_dismissed
// notification and pushes the corresponding event to the reporter.
func (s *Service) notifyReportHandled(ctx context.Context, reporterUserID, reportID, targetType, targetID, note string, resolved bool) {
	if reporterUserID == "" || s.pushService == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	notificationID := ""
	if resolved {
		id, err := s.pushService.AppendReportResolved(ctx, reporterUserID, reportID, targetType, targetID, note)
		if err != nil {
			observability.L().Error("admin.report notify report_resolved failed", "error", err, "report_id", reportID)
		} else {
			notificationID = id
		}
	} else {
		id, err := s.pushService.AppendReportDismissed(ctx, reporterUserID, reportID, targetType, targetID, note)
		if err != nil {
			observability.L().Error("admin.report notify report_dismissed failed", "error", err, "report_id", reportID)
		} else {
			notificationID = id
		}
	}
	payload := map[string]any{
		"notify_id":   notificationID,
		"report_id":   reportID,
		"target_type": targetType,
		"target_id":   targetID,
	}
	if note != "" {
		payload["note"] = note
	}
	eventType := realtime.EventTypeReportResolved
	atKey := "resolved_at"
	if !resolved {
		eventType = realtime.EventTypeReportDismissed
		atKey = "dismissed_at"
	}
	payload[atKey] = now
	realtime.DefaultHub().BroadcastToUser(reporterUserID, realtime.Event{Type: eventType, Payload: payload})
}

func (s *Service) banUser(ctx context.Context, actor AdminActor, userID, reason string, durationDays *int, appendAudit bool) (*httpx.APIError, error) {
	if strings.TrimSpace(userID) == "" || len(reason) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	// duration_days 仅拒绝负数（spec api_spec_admin.md users/set_ban：不提供或
	// 0 = 永久封禁；正数无上限约束，不引入规范外的额外上限）。
	if durationDays != nil && *durationDays < 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if actor.UserID == userID {
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "不能封禁自己"}, nil
	}
	item, found, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}, nil
	}
	_, isAdmin, err := s.repo.GetActor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isAdmin && !actor.IsSuperAdmin() {
		return httpx.Forbidden(httpx.ReasonAdminProtectedTarget, "目标为平台管理员或超管，不能封禁"), nil
	}
	duration := 0
	if durationDays != nil {
		duration = *durationDays
	}
	bannedAt := time.Now().UTC()
	state := UserBanState{Status: UserAccountStatusBanned, Reason: strings.TrimSpace(reason), DurationDays: duration, BannedAt: bannedAt}
	if err := s.repo.SetUserBanState(ctx, userID, state); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	payload := map[string]any{"reason": strings.TrimSpace(reason), "duration_days": duration, "banned_at": bannedAt.Format(time.RFC3339)}
	sessionIDs, err := s.repo.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, sessionID := range sessionIDs {
		realtime.DefaultHub().CloseSession(sessionID, realtime.Event{Type: realtime.EventTypeAccountBanned, Payload: payload})
		if err := s.repo.RevokeSession(ctx, sessionID); err != nil {
			return nil, err
		}
	}
	if appendAudit {
		if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.ban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("user"), TargetID: userID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.ban", "reason": strings.TrimSpace(reason), "duration_days": duration}}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (s *Service) appendAudit(ctx context.Context, entry AuditLogEntry) error {
	actor := AdminActor{UserID: entry.Operator.UserID}
	return s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, entry))
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

func isValidReportAction(action ReportAction) bool {
	switch action {
	case ReportActionResolve, ReportActionDismiss:
		return true
	default:
		return false
	}
}
