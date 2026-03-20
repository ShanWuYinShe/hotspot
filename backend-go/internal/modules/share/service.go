package share

import (
	"context"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, req CreateShareRequest) ShareCreated {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Create(ctx, principal.UserID, req)
}
func (s *Service) Get(ctx context.Context, shareCode string) (ShareDetail, bool) {
	return s.repo.Get(ctx, shareCode)
}
func (s *Service) List(ctx context.Context) PaginatedShares {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.List(ctx, principal.UserID)
}
func (s *Service) Revoke(ctx context.Context, shareID string) bool {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Revoke(ctx, principal.UserID, shareID)
}
