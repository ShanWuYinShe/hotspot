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
		resp.Items = append(resp.Items, AdminTagItem{TagID: "tag_" + item.Name, Name: item.Name, UsageCount: item.UsageCount, Trend: defaultTagTrend, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	if page.Next != nil {
		resp.NextCursor = encodeTagCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) DeleteTag(ctx context.Context, name string, hardDelete bool) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionTagsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	tagName := normalizeTagName(name)
	if tagName == "" {
		return httpx.InvalidRequest("参数错误"), nil
	}
	wasBlocked, hadRefs, err := s.moderationStore.GetTagDeleteState(ctx, tagName)
	if err != nil {
		return nil, err
	}
	if err := s.moderationStore.DeleteTag(ctx, tagName, hardDelete); err != nil {
		if errors.Is(err, errTagNotFound) {
			return &httpx.APIError{Code: 10300, Message: "标签不存在"}, nil
		}
		return nil, err
	}
	if !wasBlocked {
		deleteType := "soft_block"
		if hardDelete {
			deleteType = "hard_delete"
		}
		details := AuditDetails{"type": string(AuditActionTagDelete), "name": tagName, "method": deleteType}
		if hardDelete && hadRefs {
			details["warning"] = "该操作会从所有历史引用中移除标签，不可恢复"
		}
		if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionTagDelete, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeTag, TargetID: tagName, TargetName: tagName, Details: details}); err != nil {
			return nil, err
		}
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
	title := strings.TrimSpace(req.Title)
	input := systemmodule.AnnouncementCreateInput{
		Actor: systemmodule.AnnouncementActor{UserID: actor.UserID, Nickname: actor.UserID},
		Title: title, Content: strings.TrimSpace(req.Content), Severity: req.Severity,
		ActionURL: strings.TrimSpace(req.ActionURL), SendPush: req.SendPush,
	}
	item, err := s.moderationStore.CreateAnnouncement(ctx, input)
	if err != nil {
		return AnnouncementCreated{}, nil, err
	}
	if s.pushService != nil {
		usersPage, searchErr := s.repo.SearchUsers(ctx, AdminUserSearchRequest{Pagination: &PaginationParams{Limit: 1000}})
		if searchErr != nil {
			return AnnouncementCreated{}, nil, searchErr
		}
		userIDs := make([]string, 0, len(usersPage.Items))
		for _, user := range usersPage.Items {
			if user.User.UserID == "" {
				continue
			}
			userIDs = append(userIDs, user.User.UserID)
		}
		notificationID, err := s.pushService.AppendSystemAnnouncement(ctx, userIDs, item.PublicItem())
		if err != nil {
			return AnnouncementCreated{}, nil, err
		}
		if req.SendPush {
			s.pushService.PushSystemAnnouncement(item.PublicItem(), notificationID)
		}
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementCreate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: item.AnnouncementID, TargetName: title, Details: AuditDetails{"type": string(AuditActionAnnouncementCreate), "title": title}}); err != nil {
		return AnnouncementCreated{}, nil, err
	}
	return AnnouncementCreated{AnnouncementID: item.AnnouncementID}, nil, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateAnnouncementRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionAnnouncementsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if apiErr := validateAnnouncementUpdate(req); apiErr != nil {
		return apiErr, nil
	}
	current, found, err := s.moderationStore.GetAnnouncement(ctx, req.AnnouncementID)
	if err != nil {
		return nil, err
	}
	if !found || current.IsDeleted() {
		return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
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
	_, err = s.moderationStore.UpdateAnnouncement(ctx, systemmodule.AnnouncementUpdateInput{
		AnnouncementID: req.AnnouncementID,
		Title:          trimOptional(req.Title),
		Content:        trimOptional(req.Content),
		Severity:       req.Severity,
		ActionURL:      trimOptional(req.ActionURL),
	})
	if err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) {
			return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
		}
		return nil, err
	}
	details := AuditDetails{"type": string(AuditActionAnnouncementUpdate), "title": updatedTitle}
	if len(changedFields) > 0 {
		details["changed_fields"] = changedFields
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementUpdate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: req.AnnouncementID, TargetName: updatedTitle, Details: details}); err != nil {
		return nil, err
	}
	return nil, nil
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
		return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
	}
	if current.IsDeleted() {
		return nil, nil
	}
	if err := s.moderationStore.DeleteAnnouncement(ctx, announcementID); err != nil {
		if errors.Is(err, systemmodule.ErrAnnouncementNotFound()) {
			return &httpx.APIError{Code: 10300, Message: "公告不存在"}, nil
		}
		return nil, err
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementDelete, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetTypeAnnouncement, TargetID: announcementID, TargetName: current.Title, Details: AuditDetails{"type": string(AuditActionAnnouncementDelete), "title": current.Title}}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Service) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (PaginatedAdminUsers, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionUsersRead); apiErr != nil || err != nil {
		return PaginatedAdminUsers{}, apiErr, err
	}
	if err := validateAdminUserSearchRequest(req); err != nil {
		return PaginatedAdminUsers{}, err, nil
	}
	page, err := s.repo.SearchUsers(ctx, req)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAdminUsers{}, apiErr, nil
		}
		return PaginatedAdminUsers{}, nil, err
	}
	resp := PaginatedAdminUsers{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAdminUserCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) OpenAdminUser(ctx context.Context, userID string) (AdminUserItem, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionUsersRead); apiErr != nil || err != nil {
		return AdminUserItem{}, apiErr, err
	}
	if strings.TrimSpace(userID) == "" {
		return AdminUserItem{}, httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return AdminUserItem{}, apiErr, nil
		}
		return AdminUserItem{}, nil, err
	}
	if !found {
		return AdminUserItem{}, &httpx.APIError{Code: 10300, Message: "用户不存在"}, nil
	}
	return item, nil, nil
}

func (s *Service) BanUser(ctx context.Context, req AdminBanUserRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionUsersWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	return s.banUser(ctx, actor, req.UserID, req.Reason, req.DurationDays, true)
}

func (s *Service) UnbanUser(ctx context.Context, req AdminUnbanUserRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionUsersWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if strings.TrimSpace(req.UserID) == "" || len(req.Reason) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	item, found, err := s.repo.GetUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "用户不存在"}, nil
	}
	state, ok, err := s.repo.GetUserBanState(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !ok || state.Status != UserAccountStatusBanned {
		return &httpx.APIError{Code: 10400, Message: "用户未被封禁"}, nil
	}
	if _, isAdmin, err := s.repo.GetActor(ctx, req.UserID); err != nil {
		return nil, err
	} else if isAdmin && !actor.IsSuperAdmin() {
		return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.users.write"}, nil
	}
	if err := s.repo.SetUserBanState(ctx, req.UserID, UserBanState{Status: UserAccountStatusNormal}); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
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
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAdminGroups{}, apiErr, nil
		}
		return PaginatedAdminGroups{}, nil, err
	}
	resp := PaginatedAdminGroups{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAdminGroupCursor(*page.Next)
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
		return AdminGroupItem{}, &httpx.APIError{Code: 10300, Message: "群组不存在"}, nil
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
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "群组不存在"}, nil
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
		if apiErr, ok := err.(*httpx.APIError); ok {
			return PaginatedAdminReports{}, apiErr, nil
		}
		return PaginatedAdminReports{}, nil, err
	}
	resp := PaginatedAdminReports{Items: page.Items}
	if page.Next != nil {
		resp.NextCursor = encodeAdminReportCursor(*page.Next)
	}
	return resp, nil, nil
}

func (s *Service) HandleReport(ctx context.Context, req AdminReportHandleRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionReportsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	if strings.TrimSpace(req.ReportID) == "" || !isValidReportAction(req.Action) || len(req.Note) > 500 {
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
		return &httpx.APIError{Code: 10300, Message: "举报不存在"}, nil
	}
	if record.Item.Status != ReportStatusPending {
		return &httpx.APIError{Code: 10400, Message: "举报已处理"}, nil
	}
	if req.Action == ReportActionResolve {
		switch record.Item.TargetType {
		case ReportTargetTypeGroup:
			if _, _, err := s.repo.DismissGroup(ctx, record.Item.TargetID); err != nil {
				if apiErr, ok := err.(*httpx.APIError); ok {
					return apiErr, nil
				}
				return nil, err
			}
		case ReportTargetTypeUser:
			if req.BanDurationDays != nil {
				if apiErr, err := s.banUser(ctx, actor, record.TargetUser.UserID, req.Note, req.BanDurationDays, false); apiErr != nil || err != nil {
					return apiErr, err
				}
			}
		case ReportTargetTypeMoment:
			if err := s.repo.DeleteMoment(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if req.BanDurationDays != nil {
				if apiErr, err := s.banUser(ctx, actor, record.TargetUser.UserID, req.Note, req.BanDurationDays, false); apiErr != nil || err != nil {
					return apiErr, err
				}
			}
		case ReportTargetTypeComment:
			if err := s.repo.DeleteComment(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if req.BanDurationDays != nil {
				if apiErr, err := s.banUser(ctx, actor, record.TargetUser.UserID, req.Note, req.BanDurationDays, false); apiErr != nil || err != nil {
					return apiErr, err
				}
			}
		case ReportTargetTypeMessage:
			if err := s.repo.RecallGroupMessage(ctx, record.Item.TargetID); err != nil {
				if err := s.repo.RecallPrivateMessage(ctx, record.Item.TargetID); err != nil {
					return nil, err
				}
			}
			if req.BanDurationDays != nil {
				if apiErr, err := s.banUser(ctx, actor, record.TargetUser.UserID, req.Note, req.BanDurationDays, false); apiErr != nil || err != nil {
					return apiErr, err
				}
			}
		}
	}
	_, ok, err := s.repo.HandleReport(ctx, req.ReportID, req.Action, strings.TrimSpace(req.Note), &UserInfo{UserID: actor.UserID, Nickname: actor.UserID})
	if err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
			return apiErr, nil
		}
		return nil, err
	}
	if !ok {
		return &httpx.APIError{Code: 10300, Message: "举报不存在"}, nil
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("report.handle"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.UserID}, TargetType: AuditTargetType("report"), TargetID: req.ReportID, TargetName: req.ReportID, Details: map[string]any{"type": "report.handle", "report_id": req.ReportID, "action": string(req.Action), "note": strings.TrimSpace(req.Note)}}); err != nil {
		return nil, err
	}
	// Notify the reporter about the outcome (resolve/dismiss only; escalate
	// keeps the report pending and does not notify).
	if req.Action == ReportActionResolve || req.Action == ReportActionDismiss {
		s.notifyReportHandled(ctx, record.Item.Reporter.UserID, req.ReportID, string(record.Item.TargetType), record.Item.TargetID, strings.TrimSpace(req.Note), req.Action == ReportActionResolve)
	}
	return nil, nil
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
	if durationDays != nil && (*durationDays < 0 || *durationDays > 365) {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if actor.UserID == userID {
		return &httpx.APIError{Code: 10400, Message: "不能封禁自己"}, nil
	}
	item, found, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "用户不存在"}, nil
	}
	targetActor, isAdmin, err := s.repo.GetActor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		if !actor.IsSuperAdmin() {
			return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.users.write"}, nil
		}
		if targetActor.IsSuperAdmin() && !actor.IsSuperAdmin() {
			return &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: admin.users.write"}, nil
		}
	}
	duration := 0
	if durationDays != nil {
		duration = *durationDays
	}
	bannedAt := time.Now().UTC()
	state := UserBanState{Status: UserAccountStatusBanned, Reason: strings.TrimSpace(reason), DurationDays: duration, BannedAt: bannedAt}
	if err := s.repo.SetUserBanState(ctx, userID, state); err != nil {
		if apiErr, ok := err.(*httpx.APIError); ok {
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
