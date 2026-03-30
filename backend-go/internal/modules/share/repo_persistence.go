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

func (r *PersistenceRepository) Create(ctx context.Context, cmd createShareCommand) (ShareCreated, error) {
	return r.impl.Create(ctx, cmd)
}

func (r *PersistenceRepository) Get(ctx context.Context, query getShareQuery) (ShareDetail, error) {
	return r.impl.Get(ctx, query)
}

func (r *PersistenceRepository) List(ctx context.Context, query listSharesQuery) (PaginatedShares, error) {
	return r.impl.List(ctx, query)
}

func (r *PersistenceRepository) Revoke(ctx context.Context, cmd revokeShareCommand) error {
	return r.impl.Revoke(ctx, cmd)
}
