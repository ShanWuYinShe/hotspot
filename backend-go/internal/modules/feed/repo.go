package feed

import "context"

type Repository interface {
	ListMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	ListGroups(context.Context, groupsQuery) (PaginatedGroups, error)
	ListUsers(context.Context, usersQuery) (PaginatedUsers, error)
	ListTrending(context.Context, string) (TrendingBundle, error)
	Dismiss(context.Context, dismissCommand) error
	// FetchMomentImageRows loads ordered image rows for the given moment IDs
	// (used by the service-layer image hydration helper).
	FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error)
}
