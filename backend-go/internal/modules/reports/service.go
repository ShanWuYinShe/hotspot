package reports

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateReportRequest) (ReportCreated, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("reports.create failed", "error", err)
		return ReportCreated{}, err
	}
	return result, nil
}

func (s *Service) List(ctx context.Context) (PaginatedReports, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items, err := s.repo.List(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("reports.list failed", "error", err)
		return PaginatedReports{}, err
	}
	return items, nil
}
