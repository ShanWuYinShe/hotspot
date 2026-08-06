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

type GetFeedNearbyRequest struct {
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

type UserRelation struct {
	IsFollowing  bool `json:"is_following"`
	IsFollowedBy bool `json:"is_followed_by"`
	IsBlocked    bool `json:"is_blocked"`
}

type UserSocialContext struct {
	MutualFriendsCount int `json:"mutual_friends_count"`
	CommonGroupsCount  int `json:"common_groups_count,omitempty"`
}

// UserItem follows spec api_spec_types.md UserItem: privacy-controlled fields
// (presence_status/last_online_at) are omitted when the target's
// presence_visible is off; stats are returned on the detail path.
type UserItem struct {
	User           UserInfo            `json:"user"`
	Bio            string              `json:"bio,omitempty"`
	Tags           []string            `json:"tags,omitempty"`
	PresenceStatus *UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnlineAt   *string             `json:"last_online_at,omitempty"`
	Location       *RelativeLocation   `json:"location,omitempty"`
	Stats          *UserStats          `json:"stats,omitempty"`
	Relation       *UserRelation       `json:"relation,omitempty"`
	SocialContext  *UserSocialContext  `json:"social_context,omitempty"`
}

type PaginatedUsers struct {
	Items      []UserItem `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type GroupInfo struct {
	GroupID      string `json:"group_id"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	MembersCount int    `json:"members_count"`
}

// MyGroupState follows spec api_spec_types.md MyGroupState. unread_count is
// computed from group_message_read_state; is_pinned/muted_until come from the
// member's group_members row.
type MyGroupState struct {
	Role        GroupRole `json:"role"`
	IsJoined    bool      `json:"is_joined"`
	UnreadCount int       `json:"unread_count"`
	IsMuted     bool      `json:"is_muted"`
	MutedUntil  *string   `json:"muted_until,omitempty"`
	IsPinned    bool      `json:"is_pinned"`
	JoinedAt    *string   `json:"joined_at,omitempty"`
}

type GroupItem struct {
	Group              GroupInfo        `json:"group"`
	Description        string           `json:"description,omitempty"`
	Tags               []string         `json:"tags,omitempty"`
	JoinMethod         GroupJoinMethod  `json:"join_method"`
	RequiresApproval   bool             `json:"requires_approval"`
	HasPassword        bool             `json:"has_password,omitempty"`
	Location           RelativeLocation `json:"location"`
	MyState            *MyGroupState    `json:"my_state,omitempty"`
	FriendsInGroup     int              `json:"friends_in_group,omitempty"`
	LastActivityAt     *string          `json:"last_activity_at,omitempty"`
	OnlineMembersCount *int             `json:"online_members_count,omitempty"`
	NextActivity       *string          `json:"next_activity,omitempty"`
	CreatedAt          *string          `json:"created_at,omitempty"`
	OwnerID            string           `json:"owner_id,omitempty"`
	IsSearchable       bool             `json:"is_searchable,omitempty"`
	MaxMembers         int              `json:"max_members,omitempty"`
}

type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

type MyMomentState struct {
	IsFavorited bool `json:"is_favorited"`
}

type RecommendationContext = string

type MomentImage struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
}

type MomentItem struct {
	MomentID       string                 `json:"moment_id"`
	Description    string                 `json:"description"`
	Images         []MomentImage          `json:"images,omitempty"`
	Tags           []string               `json:"tags"`
	Mentions       []string               `json:"mentions,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	LikesCount     int                    `json:"likes_count"`
	CommentsCount  int                    `json:"comments_count"`
	SharesCount    int                    `json:"shares_count"`
	User           UserInfo               `json:"user"`
	Location       RelativeLocation       `json:"location"`
	Reactions      []Reaction             `json:"reactions,omitempty"`
	MyState        *MyMomentState         `json:"my_state,omitempty"`
	Recommendation *RecommendationContext `json:"recommendation,omitempty"`
}

type PaginatedMoments struct {
	Items      []MomentItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type TrendingBundle struct {
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
