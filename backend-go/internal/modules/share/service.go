package share

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateShareRequest) (ShareCreated, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("share.create failed", "error", err)
		return ShareCreated{}, err
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, shareCode string) (ShareDetail, bool, error) {
	item, found, err := s.repo.Get(ctx, shareCode)
	if err != nil {
		observability.L().Error("share.get failed", "error", err, "share_code", shareCode)
		return ShareDetail{}, false, err
	}
	return item, found, nil
}

func (s *Service) List(ctx context.Context) (PaginatedShares, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items, err := s.repo.List(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("share.list failed", "error", err)
		return PaginatedShares{}, err
	}
	return items, nil
}

func (s *Service) Revoke(ctx context.Context, shareID string) (bool, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	revoked, err := s.repo.Revoke(ctx, principal.UserID, shareID)
	if err != nil {
		observability.L().Error("share.revoke failed", "error", err, "share_id", shareID)
		return false, err
	}
	return revoked, nil
}
