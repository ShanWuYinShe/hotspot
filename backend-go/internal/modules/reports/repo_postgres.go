package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return ReportCreated{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	reportID := fmt.Sprintf("report_%d", now.UnixNano())
	if err := queryable.Exec(ctx, queryInsertReport, reportID, userID, string(req.TargetType), req.TargetID, string(req.Reason), req.Description, string(ReportStatusPending), now.Format(time.RFC3339)); err != nil {
		return ReportCreated{}, err
	}
	return ReportCreated{ReportID: reportID}, nil
}

func (r *PostgresRepository) FindExistingReport(ctx context.Context, userID string, targetType ReportTargetType, targetID string) (string, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var reportID string
	if err := queryable.QueryRow(ctx, queryFindExistingReport, string(targetType), targetID).Scan(&reportID); err != nil {
		return "", false, nil
	}
	return reportID, true, nil
}

func (r *PostgresRepository) TargetOwner(ctx context.Context, targetType ReportTargetType, targetID string) (string, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	query := queryReportTargetOwnerUser
	switch targetType {
	case ReportTargetTypeUser:
		query = queryReportTargetOwnerUser
	case ReportTargetTypeMoment:
		query = queryReportTargetOwnerMoment
	case ReportTargetTypeComment:
		query = queryReportTargetOwnerComment
	case ReportTargetTypeMessage:
		query = queryReportTargetOwnerMessage
	case ReportTargetTypeGroup:
		query = queryReportTargetOwnerGroup
	default:
		return "", false, nil
	}
	var ownerUserID string
	if err := queryable.QueryRow(ctx, query, targetID).Scan(&ownerUserID); err != nil {
		return "", false, nil
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	return ownerUserID, ownerUserID != "", nil
}
