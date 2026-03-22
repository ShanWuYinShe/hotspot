package users

import "context"

type Repository interface {
	GetProfile(context.Context, string) (MyProfile, bool, error)
	PutProfile(context.Context, MyProfile) error
	DeleteProfile(context.Context, string) error
	GetSettings(context.Context, string) (UserSettings, bool, error)
	PutSettings(context.Context, string, UserSettings) error
	SearchProfiles(context.Context, SearchUsersRequest) ([]MyProfile, error)
	Follow(context.Context, string, string) error
	Unfollow(context.Context, string, string) error
	Block(context.Context, string, string) error
	Unblock(context.Context, string, string) error
	SetRemark(context.Context, string, string, string) error
	ListBlocked(context.Context, string) ([]BlockedUserItem, error)
	Relationship(context.Context, string, string) (Relationship, error)
}
