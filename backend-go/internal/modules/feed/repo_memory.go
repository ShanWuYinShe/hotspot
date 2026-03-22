package feed

import "context"

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) ListMoments(_ context.Context) (PaginatedFeedItems, error) {
	return PaginatedFeedItems{Items: []map[string]any{{"moment_id": "moment_feed_1", "description": "feed moment"}}}, nil
}

func (r *InMemoryRepository) ListGroups(_ context.Context) (PaginatedFeedItems, error) {
	return PaginatedFeedItems{Items: []map[string]any{{"group_id": "group_feed_1", "name": "feed group"}}}, nil
}

func (r *InMemoryRepository) ListUsers(_ context.Context) (PaginatedFeedItems, error) {
	return PaginatedFeedItems{Items: []map[string]any{{"user_id": "user_feed_1", "nickname": "feed user"}}}, nil
}

func (r *InMemoryRepository) ListTrending(_ context.Context) (TrendingBundle, error) {
	return TrendingBundle{
		TrendingTags: []map[string]any{{"name": "hot", "usage_count": 1, "trend": "rising"}},
		HotMoments:   []map[string]any{{"moment_id": "moment_hot_1", "description": "hot moment"}},
	}, nil
}
