package feed

import "context"

type Repository interface {
	ListFollowingMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	ListNearbyMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	ListNearbyGroups(context.Context, groupsQuery) (PaginatedGroups, error)
	ListNearbyUsers(context.Context, usersQuery) (PaginatedUsers, error)
	ListTrending(context.Context, string) (TrendingBundle, error)
	FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error)
}
