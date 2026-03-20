package notifications

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context, req NotificationListRequest) PaginatedNotifications {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.List(ctx, principal.UserID, req)
}
func (s *Service) UnreadCount(ctx context.Context) UnreadCountResult {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCount(ctx, principal.UserID)
}
func (s *Service) MarkAsRead(ctx context.Context, ids []string) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.MarkAsRead(ctx, principal.UserID, ids)
}
func (s *Service) MarkAllRead(ctx context.Context) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	s.repo.MarkAllRead(ctx, principal.UserID)
}

func (s *Service) PushSystemAnnouncement(userID, announcementID, severity string) {
	realtime.DefaultHub().BroadcastToUser(userID, realtime.Event{Type: "SystemAnnouncement", Payload: map[string]any{"announcement_id": announcementID, "severity": severity, "created_at": time.Now().UTC().Format(time.RFC3339)}})
}
