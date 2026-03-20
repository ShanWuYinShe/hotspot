package users

import "context"

type Repository interface {
	GetProfile(context.Context, string) (MyProfile, bool)
	PutProfile(context.Context, MyProfile)
	DeleteProfile(context.Context, string)
	GetSettings(context.Context, string) (UserSettings, bool)
	PutSettings(context.Context, string, UserSettings)
	SearchProfiles(context.Context, SearchUsersRequest) []MyProfile
	Follow(context.Context, string, string)
	Unfollow(context.Context, string, string)
	Block(context.Context, string, string)
	Unblock(context.Context, string, string)
	SetRemark(context.Context, string, string, string)
	ListBlocked(context.Context, string) []BlockedUserItem
	Relationship(context.Context, string, string) Relationship
}
