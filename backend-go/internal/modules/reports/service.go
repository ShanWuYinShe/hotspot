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
	if req.TargetType == ReportTargetTypeMessage && !isValidReportSnapshot(req.MessageSnapshot) {
		// 消息举报必须携带完整快照（spec api_spec_report.md）：管理员基于
		// 快照审核，不直接查询私聊/群聊消息库。
		return ReportCreated{}, errors.New("reports: invalid message snapshot")
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
	existingID, duplicate, err := s.repo.FindExistingReport(ctx, principal.UserID, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reports.create duplicate check failed", "error", err)
		return ReportCreated{}, err
	}
	if duplicate {
		return ReportCreated{ReportID: existingID, IsDuplicate: true}, nil
	}
	result, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("reports.create failed", "error", err)
		return ReportCreated{}, err
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, req ListReportsRequest) (PaginatedReports, error) {
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

// isValidReportSnapshot 校验消息举报快照完整性（spec api_spec_types.md
// ReportSnapshot）：会话类型/会话 ID/消息类型/发送时间均须非空，
// 且会话类型限定 private|group。
func isValidReportSnapshot(snapshot *ReportSnapshot) bool {
	if snapshot == nil {
		return false
	}
	if snapshot.ChatType != "private" && snapshot.ChatType != "group" {
		return false
	}
	return snapshot.TargetChatID != "" && snapshot.MessageType != "" && snapshot.SentAt != ""
}
