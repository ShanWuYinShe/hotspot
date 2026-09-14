package users

import (
	"context"
)

type Repository interface {
	GetProfile(context.Context, string) (MyProfile, bool, error)
	// GetProfiles 批量取画像（一次往返，供 BatchGet 列表补全用；不含 Stats，
	// 需要统计的调用方走 GetProfile）。缺席/已注销判定由调用方按 IsDeleted 做。
	GetProfiles(context.Context, []string) (map[string]MyProfile, error)
	// MutualFriends 一次查询返回双向共同好友总数 + 前 previewLimit 个预览
	//（总数精确，不截断）。替代逐用户全量分页好友列表再内存求交的做法。
	MutualFriends(context.Context, string, string, int) (int, []UserInfo, error)
	PutProfile(context.Context, MyProfile) error
	DeleteProfile(context.Context, string) error
	GetSettings(context.Context, string) (UserSettings, bool, error)
	GetSettingsBatch(context.Context, []string) (map[string]UserSettings, error)
	PutSettings(context.Context, string, UserSettings) error
	SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error)
	Follow(context.Context, string, string, FollowSource, string) error
	Unfollow(context.Context, string, string) error
	Block(context.Context, string, string, string) error
	// BlockCascade 原子完成「拉黑 + 解除双方关注」级联（spec api_spec_user.md block），
	// 避免分步执行时第三次写失败留下"已解除关注但未拉黑"的中间状态。
	BlockCascade(context.Context, string, string, string) error
	Unblock(context.Context, string, string) error
	SetRemark(context.Context, string, string, string) error
	// CheckBlockedTags 返回 tags 中命中平台屏蔽标签（admin_blocked_tags）的标签名；
	// 为空表示无屏蔽标签。用于 update_account 等资料 tag 写入拒绝（10400）。
	CheckBlockedTags(context.Context, []string) ([]string, error)
	ListBlocked(context.Context, string, int, string) (PaginatedBlockedUsers, error)
	Relationship(context.Context, string, string) (Relationship, error)
	RelationshipBatch(context.Context, string, []string) (map[string]Relationship, error)
	// BlockedEitherWay 一次查询返回 a、b 之间的双向拉黑状态：
	// callerBlocked = a 拉黑了 b，targetBlocked = b 拉黑了 a。
	BlockedEitherWay(context.Context, string, string) (callerBlocked, targetBlocked bool, err error)
	ListFriends(context.Context, string, int, string) (PaginatedFriends, error)
	ListFollowing(context.Context, string, int, string) (PaginatedFollowing, error)
	ListFollowers(context.Context, string, int, string) (PaginatedFollowers, error)
	UserExists(context.Context, string) (bool, error)
}
