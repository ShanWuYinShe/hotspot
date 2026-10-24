package groups

import (
	"context"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

func (s *Service) CreateAnnouncement(ctx context.Context, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	op := observability.Begin("groups.create_announcement", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if apiErr := validateGroupAnnouncementContent(&req.Title, &req.Content); apiErr != nil {
		op.Warn("invalid request", "reason", apiErr.Message)
		return GroupAnnouncementItem{}, apiErr
	}
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		op.Fail(err)
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		op.Warn("not found")
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		op.Warn("permission denied")
		return GroupAnnouncementItem{}, ErrAccessRestricted
	}
	item, err := s.repo.CreateAnnouncement(ctx, principal.UserID, req)
	if err != nil {
		op.Fail(err)
		return GroupAnnouncementItem{}, err
	}
	// created_by 昵称统一在 service 解析（审查 P2-3）：repo 侧（尤其 PG 插入
	// 路径）不回查 users 表，此前响应里 Nickname 恒为裸 user_id。
	item.CreatedBy = s.resolveUserInfo(ctx, principal.UserID)
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{
		Type: realtime.EventTypeGroupAnnouncementCreated,
		Payload: GroupAnnouncementCreatedPayload{
			GroupID:        req.GroupID,
			AnnouncementID: item.AnnouncementID,
			Title:          item.Title,
			CreatedBy:      s.resolveUserInfo(ctx, principal.UserID),
			CreatedAt:      item.CreatedAt,
		},
	})
	op.Complete()
	return item, nil
}

func (s *Service) ListAnnouncements(ctx context.Context, req GroupAnnouncementsRequest) ([]GroupAnnouncementItem, error) {
	op := observability.Begin("groups.list_announcements", "group_id", req.GroupID)
	// 条数来自 AppConfig.response_defaults.group_announcements_count
	//（spec api_spec_group.md announcements/fetch：默认 10、硬上限 50）。
	limit := 10
	if s.appConfig != nil {
		if configured := s.appConfig.Get(ctx).ResponseDefaults.GroupAnnouncementsCount; configured > 0 {
			limit = configured
		}
	}
	items, err := s.repo.ListAnnouncements(ctx, req.GroupID, limit)
	if err != nil {
		op.Fail(err)
		return nil, err
	}
	op.Complete("count", len(items))
	return items, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	if apiErr := validateGroupAnnouncementContent(req.Title, req.Content); apiErr != nil {
		return GroupAnnouncementItem{}, apiErr
	}
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return GroupAnnouncementItem{}, ErrAccessRestricted
	}
	if err := s.repo.UpdateAnnouncement(ctx, req.GroupID, req); err != nil {
		return GroupAnnouncementItem{}, err
	}
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）：
	// 归属双键直查替代全量 List 再线性扫描找回单条（审查 P2-2）。
	item, found, err := s.repo.GetAnnouncement(ctx, req.GroupID, req.AnnouncementID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	if !found {
		return GroupAnnouncementItem{}, ErrAnnouncementNotFound
	}
	return item, nil
}

func (s *Service) DeleteAnnouncement(ctx context.Context, req DeleteGroupAnnouncementRequest) error {
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		return ErrAccessRestricted
	}
	return s.repo.DeleteAnnouncement(ctx, req.GroupID, req.AnnouncementID)
}

func (s *Service) UpdatePreferences(ctx context.Context, req UpdateGroupPreferencesRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGroupNotFound
	}
	if group.ViewerState == nil {
		return ErrNotMember
	}
	return s.repo.UpdatePreferences(ctx, principal.UserID, req)
}
