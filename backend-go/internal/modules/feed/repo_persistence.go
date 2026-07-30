package feed

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) ListFollowingMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	return r.impl.ListFollowingMoments(ctx, query)
}

func (r *PersistenceRepository) ListNearbyMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	return r.impl.ListNearbyMoments(ctx, query)
}

func (r *PersistenceRepository) ListNearbyGroups(ctx context.Context, query groupsQuery) (PaginatedGroups, error) {
	return r.impl.ListNearbyGroups(ctx, query)
}

func (r *PersistenceRepository) ListNearbyUsers(ctx context.Context, query usersQuery) (PaginatedUsers, error) {
	return r.impl.ListNearbyUsers(ctx, query)
}

func (r *PersistenceRepository) ListTrending(ctx context.Context, viewerUserID string) (TrendingBundle, error) {
	return r.impl.ListTrending(ctx, viewerUserID)
}

func (r *PersistenceRepository) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	return r.impl.FetchMomentImageRows(ctx, momentIDs)
}
