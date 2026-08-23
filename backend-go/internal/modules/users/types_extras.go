package users

import (
	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/types"
)

type Relationship struct {
	IsFollowing  bool `json:"is_following"`
	IsFollowedBy bool `json:"is_followed_by"`
	IsBlocked    bool `json:"is_blocked"`
}

var (
	RelationshipNone      = Relationship{}
	RelationshipFollowing = Relationship{IsFollowing: true}
	RelationshipBlocked   = Relationship{IsBlocked: true}
)

type RelativeLocation = types.RelativeLocation

type UserInfo = types.UserInfo

type SocialContext struct {
	MutualFriendsCount   int        `json:"mutual_friends_count"`
	MutualFriendsPreview []UserInfo `json:"mutual_friends_preview"`
	CommonGroupsCount    *int       `json:"common_groups_count"`
	CommonTags           []string   `json:"common_tags"`
}

type UserItem struct {
	User           UserInfo            `json:"user"`
	Bio            *string             `json:"bio"`
	Tags           []string            `json:"tags"`
	PresenceStatus *UserPresenceStatus `json:"presence_status"`
	LastOnline     *string             `json:"last_online_at"`
	Location       *RelativeLocation   `json:"location"`
	Stats          *UserStats          `json:"stats"`
	Relation       *Relationship       `json:"relation"`
	SocialContext  *SocialContext      `json:"social_context"`
}

type UserOpenTheDoorRequest struct {
	UserID               string `json:"user_id"`
	IncludeRecentMoments bool   `json:"include_recent_moments,omitempty"` // true 时返回主页首屏 recent_moments（spec user.md → open_the_door）
}

// BatchGetUsersRequest backs POST /api/v1/users/batch_fetch
// (spec api_spec_user.md). At most 100 user ids per call.
type BatchGetUsersRequest struct {
	UserIDs []string `json:"user_ids"`
}

// UsersBatchResult is the users/batch_fetch response (spec api_spec_user.md
// → UsersBatchResult）：users: user_id -> UserInfo map（无查看者字段的共享
// 类型，仅为补全昵称/头像等基础信息）；ids that do not exist are absent.
type UsersBatchResult struct {
	Users map[string]UserInfo `json:"users"`
}

type FollowSource string

const (
	FollowSourceMoment         FollowSource = "moment"
	FollowSourceSearch         FollowSource = "search"
	FollowSourceRecommendation FollowSource = "recommendation"
	FollowSourceProfile        FollowSource = "profile"
	FollowSourceGroup          FollowSource = "group"
)

type SearchUsersRequest struct {
	Keyword        string            `json:"keyword,omitempty"`
	Tag            string            `json:"tag,omitempty"`
	Radius         *int              `json:"radius,omitempty"`
	SortBy         string            `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint          `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type PaginatedUsers struct {
	Items      []UserItem `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type PaginatedProfiles struct {
	Items         []MyProfile
	NextCursor    string
	InvalidCursor bool
}

type UpdateLocationRequest struct {
	Location Location `json:"location"`
}

type UpgradeAccountRequest struct {
	LoginID  string `json:"login_id"`
	Password string `json:"password"`
}

type DeleteAccountRequest struct {
	Confirmation string `json:"confirmation"`
}

type FollowRequest struct {
	UserID         string       `json:"user_id"`
	Active         bool         `json:"active"`
	Source         FollowSource `json:"source,omitempty"`
	SourceMomentID string       `json:"source_moment_id,omitempty"`
}

type BlockRequest struct {
	UserID string `json:"user_id"`
	Active bool   `json:"active"`
	Reason string `json:"reason,omitempty"`
}

type SetRemarkRequest struct {
	UserID string `json:"user_id"`
	Remark string `json:"remark"`
}

type ListFriendsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// RelatedUserItem 是 fetch_friends / fetch_following / fetch_followers 的统一
// 响应项（spec api_spec_user.md → RelatedUserItem）。user 为完整 UserItem，
// 隐私字段（presence_status / last_online_at）随 user 返回；remark 仅查自己时返回。
type RelatedUserItem struct {
	User           UserItem `json:"user"`
	FollowedAt     string   `json:"followed_at"`
	FollowedBackAt *string  `json:"followed_back_at"`
	Remark         *string  `json:"remark"`
}

type PaginatedFriends struct {
	Items      []RelatedUserItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type ListFollowingRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PaginatedFollowing struct {
	Items      []RelatedUserItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type ListFollowersRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type PaginatedFollowers struct {
	Items      []RelatedUserItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type UserGroupsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type UserMomentsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type UserProfileBundle struct {
	User          UserItem                      `json:"user"`
	RecentMoments *momenttypes.PaginatedMoments `json:"recent_moments"` // 仅 include_recent_moments=true 时返回
}

type MyBlockedRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type BlockedUserItem struct {
	User      UserInfo `json:"user"`
	BlockedAt string   `json:"blocked_at"`
	Reason    *string  `json:"reason"`
}

type PaginatedBlockedUsers struct {
	Items      []BlockedUserItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type UserListGroupsResponse = grouptypes.PaginatedGroups

type UserListMomentsResponse = momenttypes.PaginatedMoments
