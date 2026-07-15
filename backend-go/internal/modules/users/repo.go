package users

import (
	"context"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
)

type Repository interface {
	GetProfile(context.Context, string) (MyProfile, bool, error)
	PutProfile(context.Context, MyProfile) error
	DeleteProfile(context.Context, string) error
	GetSettings(context.Context, string) (UserSettings, bool, error)
	GetSettingsBatch(context.Context, []string) (map[string]UserSettings, error)
	PutSettings(context.Context, string, UserSettings) error
	SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error)
	Follow(context.Context, string, string, FollowSource, string) error
	Unfollow(context.Context, string, string) error
	Block(context.Context, string, string, string) error
	Unblock(context.Context, string, string) error
	SetRemark(context.Context, string, string, string) error
	ListBlocked(context.Context, string, int, string) (PaginatedBlockedUsers, error)
	ListLikedMoments(context.Context, string, int, string, string) (momenttypes.PaginatedMoments, error)
	Relationship(context.Context, string, string) (Relationship, error)
	ListFriends(context.Context, string, int, string) (PaginatedFriends, error)
	ListFollowing(context.Context, string, int, string) (PaginatedFollowing, error)
	ListFollowers(context.Context, string, int, string) (PaginatedFollowers, error)
	ListGroups(context.Context, string, int, string) (grouptypes.PaginatedGroups, error)
	ListMoments(context.Context, string, int, string, bool) (momenttypes.PaginatedMoments, error)
	CommonGroupCount(context.Context, string, string) (int, error)
	UserExists(context.Context, string) (bool, error)
}
