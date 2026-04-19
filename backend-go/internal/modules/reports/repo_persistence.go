package reports

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) List(ctx context.Context, userID string, req MyReportsRequest) (PaginatedReports, error) {
	return r.impl.List(ctx, userID, req)
}

func (r *PersistenceRepository) ExistsOpenReport(ctx context.Context, targetType ReportTargetType, targetID string) (bool, error) {
	return r.impl.ExistsOpenReport(ctx, targetType, targetID)
}

func (r *PersistenceRepository) TargetOwner(ctx context.Context, targetType ReportTargetType, targetID string) (string, bool, error) {
	return r.impl.TargetOwner(ctx, targetType, targetID)
}
