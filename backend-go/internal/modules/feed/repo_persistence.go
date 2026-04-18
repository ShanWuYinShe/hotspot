package feed

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

func (r *PersistenceRepository) ListMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	return r.impl.ListMoments(ctx, query)
}

func (r *PersistenceRepository) ListGroups(ctx context.Context, query groupsQuery) (PaginatedGroups, error) {
	return r.impl.ListGroups(ctx, query)
}

func (r *PersistenceRepository) ListUsers(ctx context.Context, query usersQuery) (PaginatedUsers, error) {
	return r.impl.ListUsers(ctx, query)
}

func (r *PersistenceRepository) ListTrending(ctx context.Context, viewerUserID string) (TrendingBundle, error) {
	return r.impl.ListTrending(ctx, viewerUserID)
}

func (r *PersistenceRepository) Dismiss(ctx context.Context, cmd dismissCommand) error {
	return r.impl.Dismiss(ctx, cmd)
}
