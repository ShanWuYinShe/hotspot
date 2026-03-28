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

func (s *Service) Exists(ctx context.Context, momentID string) (bool, error) {
	return s.repo.Exists(ctx, momentID)
}

func (s *Service) IncrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementCommentCount(ctx, momentID)
}

func (s *Service) DecrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.DecrementCommentCount(ctx, momentID)
}

func (s *Service) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	return s.repo.Search(ctx, req)
}

func (s *Service) Update(ctx context.Context, req EditMomentRequest) (MomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, req.MomentID)
	if !ok {
		if err != nil {
			return MomentItem{}, err
		}
		return MomentItem{}, ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return MomentItem{}, ErrPermissionDenied
	}
	if err := s.repo.Update(ctx, principal.UserID, req); err != nil {
		return MomentItem{}, err
	}
	updated, _, err := s.repo.Get(ctx, req.MomentID)
	if err != nil {
		return MomentItem{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, momentID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return ErrPermissionDenied
	}
	return s.repo.Delete(ctx, momentID)
}

func (s *Service) Like(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return s.repo.Like(ctx, principal.UserID, momentID)
}

func (s *Service) Unlike(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unlike(ctx, principal.UserID, momentID)
}

func (s *Service) Favorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return s.repo.Favorite(ctx, principal.UserID, momentID)
}

func (s *Service) Unfavorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unfavorite(ctx, principal.UserID, momentID)
}

func (s *Service) ListLikes(ctx context.Context, req ListLikesRequest) ([]LikeItem, string, error) {
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > 100 {
		limit = 100
	}
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListLikes(ctx, req.MomentID, limit, cursor)
}

func (s *Service) ListFavorites(ctx context.Context, req MyFavoritesRequest) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > 100 {
		limit = 100
	}
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListFavorites(ctx, principal.UserID, limit, cursor)
}
