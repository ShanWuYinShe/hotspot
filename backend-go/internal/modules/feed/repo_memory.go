package feed

import "context"

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) ListFollowingMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

func (r *InMemoryRepository) ListNearbyMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

func (r *InMemoryRepository) ListNearbyGroups(_ context.Context, _ groupsQuery) (PaginatedGroups, error) {
	return PaginatedGroups{}, nil
}

func (r *InMemoryRepository) ListNearbyUsers(_ context.Context, _ usersQuery) (PaginatedUsers, error) {
	return PaginatedUsers{}, nil
}

func (r *InMemoryRepository) ListTrending(_ context.Context, _ string) (TrendingBundle, error) {
	return TrendingBundle{}, nil
}

func (r *InMemoryRepository) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}
