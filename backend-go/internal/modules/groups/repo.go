package groups

import "context"

type Repository interface {
	Create(context.Context, string, CreateGroupRequest) (GroupItem, error)
	Get(context.Context, string) (GroupItem, bool, error)
	Search(context.Context, string) (PaginatedGroups, error)
	Update(context.Context, string, UpdateGroupRequest) error
	Dismiss(context.Context, string) error
	ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
	SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error)
}
