package feed

import (
	"errors"
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/types"
)

type GeoPoint = types.GeoPoint

type RelativeLocation = types.RelativeLocation

type PaginationParams = types.PaginationParams

type GetFeedFollowingRequest struct {
	ReferencePoint *GeoPoint         `json:"reference_point,omitempty"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type NearbyMomentsRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type GetFeedNearbyGroupsRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type GetFeedNearbyUsersRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type UserInfo = types.UserInfo

// 通用领域类型别名（规范定义于 users/groups 模块，spec api_spec_types.md）
type UserPresenceStatus = usersmodule.UserPresenceStatus

type UserStats = usersmodule.UserStats

type GroupRole = groupsmodule.GroupRole

type GroupJoinMethod = groupsmodule.GroupJoinMethod

// UserItem / UserRelation / UserSocialContext 与 users 模块同构
// （spec api_spec_types.md：查看者状态与隐私字段恒定 nullable，null = 隐藏/不适用）。
type UserItem = usersmodule.UserItem

type UserRelation = usersmodule.Relationship

type UserSocialContext = usersmodule.SocialContext

type PaginatedUsers struct {
	Items      []UserItem `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type GroupInfo struct {
	GroupID      string  `json:"group_id"`
	Name         string  `json:"name"`
	AvatarURL    *string `json:"avatar_url"`
	MembersCount int     `json:"members_count"`
}

// GroupViewerState 与 groups 模块同构（spec api_spec_types.md）：
// viewer_state 非 null 即群成员；joined_at 恒为字符串。
type GroupViewerState = groupsmodule.GroupViewerState

// GroupItem 遵循 spec api_spec_types.md：恒定 nullable 字段恒在（null = 匿名/不适用）；
// 群管理字段（has_password/is_searchable/max_members）不在本类型——见 groups/admin/settings。
type GroupItem struct {
	Group              GroupInfo         `json:"group"`
	Description        *string           `json:"description"`
	Tags               []string          `json:"tags"`
	JoinMethod         GroupJoinMethod   `json:"join_method"`
	RequiresApproval   bool              `json:"requires_approval"`
	Location           *RelativeLocation `json:"location"`
	ViewerState        *GroupViewerState `json:"viewer_state"`
	FriendsInGroup     *int              `json:"friends_in_group"`
	LastActivityAt     *string           `json:"last_activity_at"`
	OnlineMembersCount *int              `json:"online_members_count"`
	NextActivityAt     *string           `json:"next_activity_at"`
	CreatedAt          *string           `json:"created_at"`
	OwnerID            *string           `json:"owner_id"`
}

type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}

type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

type RecommendationContext = string

type MomentImage struct {
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	MimeType     *string `json:"mime_type"`
}

// MomentItem 遵循 spec api_spec_types.md 公开固定形状：数组字段恒在（空数组 = 无），
// is_favorited 恒为布尔（匿名/未收藏为 false），location/recommendation 恒定 nullable。
type MomentItem struct {
	MomentID    string        `json:"moment_id"`
	Description string        `json:"description"`
	Images      []MomentImage `json:"images"`
	Tags        []string      `json:"tags"`
	Mentions    []string      `json:"mentions"`
	CreatedAt   string        `json:"created_at"`
	// LikesCount 兼容保留（契约已删 likes_count），不对外序列化。
	LikesCount     int                    `json:"-"`
	CommentsCount  int                    `json:"comments_count"`
	SharesCount    int                    `json:"shares_count"`
	User           UserInfo               `json:"user"`
	IsDeleted      bool                   `json:"is_deleted"`
	Location       *RelativeLocation      `json:"location"`
	Reactions      []Reaction             `json:"reactions"`
	IsFavorited    bool                   `json:"is_favorited"`
	Recommendation *RecommendationContext `json:"recommendation"`
}

type PaginatedMoments struct {
	Items      []MomentItem `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

type TrendingResult struct {
	HotMoments []MomentItem `json:"hot_moments"`
}

type momentsQuery struct {
	ViewerUserID   string
	ReferencePoint GeoPoint
	Radius         int
	Limit          int
	Cursor         string
}

type usersQuery struct {
	ViewerUserID   string
	ReferencePoint GeoPoint
	Radius         int
	Limit          int
	Cursor         string
}

type groupsQuery struct {
	ViewerUserID   string
	ReferencePoint GeoPoint
	Radius         int
	Limit          int
	Cursor         string
}

type itemCursor struct {
	Score     int    `json:"score,omitempty"`
	Distance  int    `json:"distance,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	ID        string `json:"id"`
}

type momentCandidate struct {
	Item      MomentItem
	Score     int
	Distance  int
	CreatedAt time.Time
}

type userCandidate struct {
	Item     UserItem
	Distance int
}

type groupCandidate struct {
	Item     GroupItem
	Distance int
}

var ErrInvalidFeedRequest = errors.New("feed: invalid request")
var ErrInvalidCursor = errors.New("feed: invalid cursor")

// nonNilTags 保证 tags 等恒定数组字段序列化为 [] 而非 null（spec 恒定形状）。
func nonNilTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
