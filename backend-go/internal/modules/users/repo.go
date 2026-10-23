package users

import (
	"context"
)

// ProfilePatch update_account 的列级增量（审查 P1-3）：仅更新非 nil 字段，
// 其余列不动——替代「读全量→改→全量回写」路径（该路径并发更新互相丢写，
// 且会用读取时刻的快照覆写 presence_status/last_online 等运行时字段）。
type ProfilePatch struct {
	Nickname  *string
	AvatarURL *string
	Bio       *string
	Gender    *string
}

type Repository interface {
	GetProfile(context.Context, string) (MyProfile, bool, error)
	// GetProfiles 批量取画像（一次往返，供 BatchGet 列表补全用；不含 Stats，
	// 需要统计的调用方走 GetProfile）。缺席/已注销判定由调用方按 IsDeleted 做。
	GetProfiles(context.Context, []string) (map[string]MyProfile, error)
	// MutualFriends 一次查询返回双向共同好友总数 + 前 previewLimit 个预览
	//（总数精确，不截断）。替代逐用户全量分页好友列表再内存求交的做法。
	MutualFriends(context.Context, string, string, int) (int, []UserInfo, error)
	PutProfile(context.Context, MyProfile) error
	// UpdateProfile 增量更新资料：只写 patch 中非 nil 的列；tags 非 nil 时
	// 在同一事务内整组替换标签（tags==nil 表示本次不动标签）。返回目标行
	// 是否存在（false 时调用方走默认建档路径，与旧全量回写行为一致）。
	UpdateProfile(context.Context, string, ProfilePatch, *[]string) (bool, error)
	// UpdateProfileLocation 只更新位置列（含 s2_cell_id 重算），不触碰
	// presence 等运行时列（WS 生命周期维护）。
	UpdateProfileLocation(context.Context, string, Location) (bool, error)
	DeleteProfile(context.Context, string) error
	GetSettings(context.Context, string) (UserSettings, bool, error)
	GetSettingsBatch(context.Context, []string) (map[string]UserSettings, error)
	PutSettings(context.Context, string, UserSettings) error
	// UpdateSettings 增量更新设置：只写 patch 中非 nil 的列（旧全量回写
	// 会并发丢字段，审查 P1-3）。返回设置行是否存在。
	UpdateSettings(context.Context, string, PartialUserSettings) (bool, error)
	SearchProfiles(context.Context, SearchUsersRequest) (PaginatedProfiles, error)
	// Follow 建立关注关系，返回本次是否真的发生了「未关注 → 关注」状态迁移。
	// 已关注时为 no-op 返回 (false, nil)，调用方据此跳过重复通知（审计 F-12）。
	Follow(context.Context, string, string, FollowSource, string) (changed bool, err error)
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
	// NearbyMomentNotifyTargets 返回以 (centerLat, centerLng) 为中心、
	// radiusMeters 半径内、开启了 nearby_moment_notify 且带有最新位置的用户
	//（排除 excludeUserID，通常为瞬间作者）。供 moments 附近新瞬间通知用。
	NearbyMomentNotifyTargets(context.Context, float64, float64, int, string) ([]NearbyNotifyTarget, error)
	UserExists(context.Context, string) (bool, error)
}
