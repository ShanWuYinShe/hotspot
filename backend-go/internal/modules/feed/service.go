package feed

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ListMoments(ctx context.Context) PaginatedFeedItems {
	_ = ctx
	return PaginatedFeedItems{Items: []map[string]any{{"moment_id": "moment_feed_1", "description": "feed moment"}}}
}

func (s *Service) ListGroups(ctx context.Context) PaginatedFeedItems {
	_ = ctx
	return PaginatedFeedItems{Items: []map[string]any{{"group_id": "group_feed_1", "name": "feed group"}}}
}

func (s *Service) ListUsers(ctx context.Context) PaginatedFeedItems {
	_ = ctx
	return PaginatedFeedItems{Items: []map[string]any{{"user_id": "user_feed_1", "nickname": "feed user"}}}
}

func (s *Service) ListTrending(ctx context.Context) TrendingBundle {
	_ = ctx
	return TrendingBundle{TrendingTags: []map[string]any{{"name": "hot", "usage_count": 1, "trend": "rising"}}, HotMoments: []map[string]any{{"moment_id": "moment_hot_1", "description": "hot moment"}}}
}
