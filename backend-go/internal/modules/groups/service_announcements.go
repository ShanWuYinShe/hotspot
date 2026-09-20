package groups

import (
	"context"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"time"
)

func (s *Service) CreateAnnouncement(ctx context.Context, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.create_announcement started", "group_id", req.GroupID)
	principal, _ := platformauth.PrincipalFromContext(ctx)
	group, ok, err := s.repo.Get(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.create_announcement failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, err
	}
	if !ok {
		observability.L().Warn("groups.create_announcement not found", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, ErrGroupNotFound
	}
	if group.ViewerState == nil || (group.ViewerState.Role != GroupRoleOwner && group.ViewerState.Role != GroupRoleAdmin) {
		observability.L().Warn("groups.create_announcement permission denied", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, ErrAccessRestricted
	}
	item, err := s.repo.CreateAnnouncement(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("groups.create_announcement failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return GroupAnnouncementItem{}, err
	}
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
	observability.L().Info("groups.create_announcement completed", "group_id", req.GroupID, "elapsed_ms", time.Since(startedAt).Milliseconds())
	return item, nil
}

func (s *Service) ListAnnouncements(ctx context.Context, req GroupAnnouncementsRequest) ([]GroupAnnouncementItem, error) {
	startedAt := time.Now()
	observability.L().Info("groups.list_announcements started", "group_id", req.GroupID)
	items, err := s.repo.ListAnnouncements(ctx, req.GroupID)
	if err != nil {
		observability.L().Error("groups.list_announcements failed", "error", err, "elapsed_ms", time.Since(startedAt).Milliseconds())
		return nil, err
	}
	observability.L().Info("groups.list_announcements completed", "group_id", req.GroupID, "count", len(items), "elapsed_ms", time.Since(startedAt).Milliseconds())
	return items, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, req UpdateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
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
	// 内容类更新返回更新后完整对象（spec api_spec_base.md「设计规范」）
	items, err := s.repo.ListAnnouncements(ctx, req.GroupID)
	if err != nil {
		return GroupAnnouncementItem{}, err
	}
	for _, item := range items {
		if item.AnnouncementID == req.AnnouncementID {
			return item, nil
		}
	}
	return GroupAnnouncementItem{}, ErrAnnouncementNotFound
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
