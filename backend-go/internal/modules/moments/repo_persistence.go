package moments

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

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) Get(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return r.impl.Get(ctx, momentID)
}

func (r *PersistenceRepository) ListMine(ctx context.Context, userID string) (PaginatedMoments, error) {
	return r.impl.ListMine(ctx, userID)
}

func (r *PersistenceRepository) IncrementCommentCount(ctx context.Context, momentID string) error {
	return r.impl.IncrementCommentCount(ctx, momentID)
}
