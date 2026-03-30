package feed

import "context"

type Repository interface {
	ListMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	ListGroups(context.Context, groupsQuery) (PaginatedGroups, error)
	ListUsers(context.Context, usersQuery) (PaginatedUsers, error)
	ListTrending(context.Context, string) (TrendingBundle, error)
}
