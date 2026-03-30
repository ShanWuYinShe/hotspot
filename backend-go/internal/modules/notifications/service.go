package notifications

import (
	"context"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, req NotificationListRequest) (PaginatedNotifications, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	normalizedReq := NotificationListRequest{Type: query.Type, Pagination: &PaginationParams{Limit: query.Limit}}
	if query.Cursor != nil {
		normalizedReq.Pagination.Cursor = encodeNotificationCursor(*query.Cursor)
	}
	items, err := s.repo.List(ctx, principal.UserID, normalizedReq)
	if err != nil {
		observability.L().Error("notifications.list failed", "error", err)
		return PaginatedNotifications{}, err
	}
	return items, nil
}

func (s *Service) UnreadCount(ctx context.Context) (UnreadCountResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.UnreadCount(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("notifications.unread_count failed", "error", err)
		return UnreadCountResult{}, err
	}
	return result, nil
}

func (s *Service) MarkAsRead(ctx context.Context, ids []string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAsRead(ctx, principal.UserID, ids); err != nil {
		observability.L().Error("notifications.mark_as_read failed", "error", err, "count", len(ids))
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("notifications.mark_all_read failed", "error", err)
		return err
	}
	return nil
}

func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem) {
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: "SystemAnnouncement", Payload: item})
}
