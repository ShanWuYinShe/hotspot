package admin

import (
	"context"
	"errors"
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/httpx"
	"strings"
	"time"
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
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionTagSetBan, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetTypeTag, TargetID: tagName, TargetName: tagName, Details: details}); err != nil {
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
		Actor: systemmodule.AnnouncementActor{UserID: actor.UserID, Nickname: actor.Nickname},
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
		// 整条公告共用一个 notify_id（跨批次复用）：契约要求 WS 的 notify_id
		// 与 notify/fetch 的主键一致，客户端据此去重与 mark_read。若每批各自
		// 生成 id，只有最后一批能被客户端命中（审计 F-06）。
		notificationID := s.pushService.NewAnnouncementNotifyID()
		for {
			batch, listErr := s.repo.ListUserIDsAfter(ctx, afterUserID, fanoutBatch)
			if listErr != nil {
				return AdminAnnouncementItem{}, nil, listErr
			}
			if len(batch) == 0 {
				break
			}
			if _, err := s.pushService.AppendSystemAnnouncementWithID(ctx, notificationID, batch, item.PublicItem()); err != nil {
				return AdminAnnouncementItem{}, nil, err
			}
			if len(batch) < fanoutBatch {
				break
			}
			afterUserID = batch[len(batch)-1]
		}
		// BroadcastToAll 是一次全体在线广播，必须移出批次循环（否则广播 N 次）。
		if req.SendPush && notificationID != "" {
			s.pushService.PushSystemAnnouncement(item.PublicItem(), notificationID)
		}
	}
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementCreate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetTypeAnnouncement, TargetID: item.AnnouncementID, TargetName: title, Details: AuditDetails{"type": string(AuditActionAnnouncementCreate), "title": title}}); err != nil {
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
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementUpdate, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetTypeAnnouncement, TargetID: req.AnnouncementID, TargetName: updatedTitle, Details: details}); err != nil {
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
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditActionAnnouncementDelete, Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetTypeAnnouncement, TargetID: announcementID, TargetName: current.Title, Details: AuditDetails{"type": string(AuditActionAnnouncementDelete), "title": current.Title}}); err != nil {
		return nil, err
	}
	return nil, nil
}
