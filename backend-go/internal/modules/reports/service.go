package reports

import (
	"context"
	"errors"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/types"
)

type Service struct {
	repo             Repository
	messageSnapshots MessageSnapshotSource // optional; nil 时消息举报按目标不存在处理
}

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

// MessageSnapshotSource 基于消息真值生成举报快照的最小依赖接口（spec
// api_spec_types.md ReportSnapshot 生成方为服务端）。由 bootstrap 接线
// 私聊/群聊两个消息源实现；reports 模块不直接依赖聊天模块。
type MessageSnapshotSource interface {
	// GetMessageSnapshot 按 message_id 读取消息真值并生成快照；
	// found=false 表示消息不存在（举报受理返回 10300）。
	GetMessageSnapshot(ctx context.Context, messageID string) (snapshot *ReportSnapshot, found bool, err error)
}

// SetMessageSnapshotSource 接入消息快照源（bootstrap 在两个聊天模块
// 构造完成后调用，打破构造顺序依赖）。
func (s *Service) SetMessageSnapshotSource(src MessageSnapshotSource) { s.messageSnapshots = src }

func (s *Service) Create(ctx context.Context, req CreateReportRequest) (ReportCreated, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if !isValidReportTargetType(req.TargetType) || !isValidReportReason(req.Reason) || req.TargetID == "" || (req.Description != nil && len(*req.Description) > 1000) {
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
	// 消息举报的服务端快照（spec api_spec_types.md ReportSnapshot 生成方）：
	// 基于消息真值生成并冻结，客户端不提交快照；消息不存在按目标不存在 10300。
	var snapshot *ReportSnapshot
	if req.TargetType == ReportTargetTypeMessage {
		if s.messageSnapshots == nil {
			observability.L().Error("reports.create message snapshot source unavailable", "target_id", req.TargetID)
			return ReportCreated{}, errors.New("reports: target not found")
		}
		snapshot, found, err = s.messageSnapshots.GetMessageSnapshot(ctx, req.TargetID)
		if err != nil {
			observability.L().Error("reports.create message snapshot failed", "error", err, "target_id", req.TargetID)
			return ReportCreated{}, err
		}
		if !found {
			return ReportCreated{}, errors.New("reports: target not found")
		}
	}
	existingID, duplicate, err := s.repo.FindExistingReport(ctx, principal.UserID, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reports.create duplicate check failed", "error", err)
		return ReportCreated{}, err
	}
	if duplicate {
		return ReportCreated{ReportID: existingID, IsDuplicate: true}, nil
	}
	result, err := s.repo.Create(ctx, principal.UserID, req, snapshot)
	if err != nil {
		observability.L().Error("reports.create failed", "error", err)
		return ReportCreated{}, err
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, req ListReportsRequest) (PaginatedReports, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// 分页 limit 兜底裁剪（api_spec_base.md：default=20 max=100，service 层统一归一）。
	if req.Pagination != nil {
		req.Pagination.Limit = types.NormalizeLimit(req.Pagination.Limit)
	}
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
