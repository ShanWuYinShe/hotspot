package feed

import "context"

type Repository interface {
	ListMoments(context.Context) (PaginatedFeedItems, error)
	ListGroups(context.Context) (PaginatedFeedItems, error)
	ListUsers(context.Context) (PaginatedFeedItems, error)
	ListTrending(context.Context) (TrendingBundle, error)
}
