package reports

import (
	"context"
	"errors"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, req CreateReportRequest) (ReportCreated, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if !isValidReportTargetType(req.TargetType) || !isValidReportReason(req.Reason) || req.TargetID == "" || len(req.Description) > 1000 {
		return ReportCreated{}, errors.New("reports: invalid request")
	}
	if req.TargetType == ReportTargetTypeUser && req.TargetID == principal.UserID {
		return ReportCreated{}, errors.New("reports: self report")
	}
	ownerUserID, found, err := s.repo.TargetOwner(ctx, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reports.create target lookup failed", "error", err)
		return ReportCreated{}, err
	}
	if !found {
		return ReportCreated{}, errors.New("reports: target not found")
	}
	if ownerUserID != "" && ownerUserID == principal.UserID {
		return ReportCreated{}, errors.New("reports: self report")
	}
	openExists, err := s.repo.ExistsOpenReport(ctx, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reports.create duplicate check failed", "error", err)
		return ReportCreated{}, err
	}
	if openExists {
		return ReportCreated{}, errors.New("reports: duplicate report")
	}
	result, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("reports.create failed", "error", err)
		return ReportCreated{}, err
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, req MyReportsRequest) (PaginatedReports, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items, err := s.repo.List(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("reports.list failed", "error", err)
		return PaginatedReports{}, err
	}
	return items, nil
}

func isValidReportTargetType(value ReportTargetType) bool {
	switch value {
	case ReportTargetTypeUser, ReportTargetTypeMoment, ReportTargetTypeComment, ReportTargetTypeMessage, ReportTargetTypeGroup:
		return true
	default:
		return false
	}
}

func isValidReportReason(value ReportReason) bool {
	switch value {
	case ReportReasonSpam, ReportReasonHarassment, ReportReasonViolence, ReportReasonHateSpeech, ReportReasonSexualContent, ReportReasonMisinformation, ReportReasonImpersonation, ReportReasonSelfHarm, ReportReasonOther:
		return true
	default:
		return false
	}
}
