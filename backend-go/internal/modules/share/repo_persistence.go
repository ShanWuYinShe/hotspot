package share

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

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateShareRequest) (ShareCreated, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) Get(ctx context.Context, shareCode string) (ShareDetail, bool, error) {
	return r.impl.Get(ctx, shareCode)
}

func (r *PersistenceRepository) List(ctx context.Context, userID string) (PaginatedShares, error) {
	return r.impl.List(ctx, userID)
}

func (r *PersistenceRepository) Revoke(ctx context.Context, userID, shareID string) (bool, error) {
	return r.impl.Revoke(ctx, userID, shareID)
}
