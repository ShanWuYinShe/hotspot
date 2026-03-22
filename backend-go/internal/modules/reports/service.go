package reports

import (
	"context"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateReportRequest) (ReportCreated, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Create(ctx, principal.UserID, req)
}

func (s *Service) List(ctx context.Context) (PaginatedReports, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.List(ctx, principal.UserID)
}
