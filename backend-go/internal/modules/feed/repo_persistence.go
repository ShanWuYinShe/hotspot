package feed

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory, nil)}
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
