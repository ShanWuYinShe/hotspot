package reports

import (
	"context"
	"fmt"
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
	if err := queryable.Exec(ctx, queryInsertReport, reportID, userID, req.TargetType, req.TargetID, req.Reason, req.Description, "pending", now.Format(time.RFC3339)); err != nil {
		return ReportCreated{}, err
	}
	return ReportCreated{ReportID: reportID}, nil
}
