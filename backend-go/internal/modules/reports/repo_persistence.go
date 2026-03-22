package reports

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) List(ctx context.Context, userID string) (PaginatedReports, error) {
	return r.impl.List(ctx, userID)
}
