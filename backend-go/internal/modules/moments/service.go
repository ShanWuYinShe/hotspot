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

func (s *Service) Create(ctx context.Context, req CreateMomentRequest) (MomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		return MomentItem{}, err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: "MomentCommented", Payload: map[string]any{"moment_id": item.MomentID, "moment_summary": map[string]any{"moment_id": item.MomentID, "description_preview": item.Description}, "comment_id": "created", "comment_preview": item.Description}})
	return item, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return s.repo.Get(ctx, momentID)
}

func (s *Service) ListMine(ctx context.Context) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.ListMine(ctx, principal.UserID)
}

func (s *Service) IncrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementCommentCount(ctx, momentID)
}
