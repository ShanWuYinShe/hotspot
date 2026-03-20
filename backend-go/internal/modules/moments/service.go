package moments

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	repo Repository
}

func NewService() *Service {
	return NewServiceWithRepo(NewInMemoryRepository())
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateMomentRequest) MomentItem {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item := s.repo.Create(ctx, principal.UserID, req)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: "MomentCommented", Payload: map[string]any{"moment_id": item.MomentID, "moment_summary": map[string]any{"moment_id": item.MomentID, "description_preview": item.Description}, "comment_id": "created", "comment_preview": item.Description}})
	return item
}

func (s *Service) OpenTheDoor(ctx context.Context, momentID string) (MomentItem, bool) {
	return s.repo.Get(ctx, momentID)
}

func (s *Service) ListMine(ctx context.Context) PaginatedMoments {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.ListMine(ctx, principal.UserID)
}

func (s *Service) IncrementCommentCount(momentID string) {
	s.repo.IncrementCommentCount(context.Background(), momentID)
}
