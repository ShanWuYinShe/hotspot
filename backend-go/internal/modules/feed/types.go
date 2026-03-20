package feed

type PaginatedFeedItems struct {
	Items      []map[string]any `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type TrendingBundle struct {
	TrendingTags []map[string]any `json:"trending_tags"`
	HotMoments   []map[string]any `json:"hot_moments"`
}
