package groups

import "context"

type Repository interface {
	Create(context.Context, string, CreateGroupRequest) GroupItem
	Get(context.Context, string) (GroupItem, bool)
	Search(context.Context, string) PaginatedGroups
}
