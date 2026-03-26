package notifications

import (
	"context"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, req NotificationListRequest) (PaginatedNotifications, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.List(ctx, principal.UserID, req)
}

func (s *Service) UnreadCount(ctx context.Context) (UnreadCountResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCount(ctx, principal.UserID)
}

func (s *Service) MarkAsRead(ctx context.Context, ids []string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.MarkAsRead(ctx, principal.UserID, ids)
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.MarkAllRead(ctx, principal.UserID)
}

func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem) {
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: "SystemAnnouncement", Payload: item})
}
