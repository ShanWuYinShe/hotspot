package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
	}
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

func (r *PostgresRepository) ExistsOpenReport(ctx context.Context, targetType ReportTargetType, targetID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ExistsOpenReport(ctx, targetType, targetID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var one int
	if err := queryable.QueryRow(ctx, queryExistsOpenReport, string(targetType), targetID).Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

func (r *PostgresRepository) TargetOwner(ctx context.Context, targetType ReportTargetType, targetID string) (string, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.TargetOwner(ctx, targetType, targetID)
	}
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
