package feed

import (
	"errors"
	"strings"
	"time"
)

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RelativeLocation struct {
	Point          GeoPoint `json:"point"`
	LocationName   string   `json:"location_name"`
	Distance       int      `json:"distance,omitempty"`
	BearingDegrees int      `json:"bearing_degrees,omitempty"`
}

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListFeedMomentsRequest struct {
	Source         string            `json:"source"`
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type ListFeedGroupsRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type ListFeedUsersRequest struct {
	Radius         *int              `json:"radius,omitempty"`
	ReferencePoint *GeoPoint         `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type UserRelation struct {
	IsFollowing  bool `json:"is_following"`
	IsFollowedBy bool `json:"is_followed_by"`
	IsBlocked    bool `json:"is_blocked"`
}

type UserSocialContext struct {
	MutualFriendsCount int `json:"mutual_friends_count"`
	CommonGroupsCount  int `json:"common_groups_count,omitempty"`
}

type UserItem struct {
	User          UserInfo           `json:"user"`
	Bio           string             `json:"bio,omitempty"`
	Location      *RelativeLocation  `json:"location,omitempty"`
	Relation      *UserRelation      `json:"relation,omitempty"`
	SocialContext *UserSocialContext `json:"social_context,omitempty"`
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

type MyGroupState struct {
	IsJoined bool `json:"is_joined"`
}

type GroupItem struct {
	Group          GroupInfo        `json:"group"`
	Description    string           `json:"description,omitempty"`
	HasPassword    bool             `json:"has_password,omitempty"`
	Location       RelativeLocation `json:"location"`
	MyState        *MyGroupState    `json:"my_state,omitempty"`
	FriendsInGroup int              `json:"friends_in_group,omitempty"`
	LastActivityAt string           `json:"last_activity_at,omitempty"`
}

type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type MyMomentState struct {
	IsLiked     bool `json:"is_liked"`
	IsFavorited bool `json:"is_favorited"`
}

type RecommendationContext struct {
	Type string `json:"type"`
}

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
	MyState        *MyMomentState         `json:"my_state,omitempty"`
	Recommendation *RecommendationContext `json:"recommendation,omitempty"`
}

type PaginatedMoments struct {
	Items      []MomentItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type TrendingTagItem struct {
	Name             string      `json:"name"`
	UsageCount       int         `json:"usage_count"`
	Trend            string      `json:"trend"`
	HotMomentPreview *MomentItem `json:"hot_moment_preview,omitempty"`
}

type TrendingBundle struct {
	TrendingTags []TrendingTagItem `json:"trending_tags"`
	HotMoments   []MomentItem      `json:"hot_moments"`
}

type DismissFeedItemRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

type dismissCommand struct {
	ViewerUserID string
	TargetType   string
	TargetID     string
}

type momentsQuery struct {
	ViewerUserID   string
	Source         string
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

func normalizeDismissTargetType(value string) string {
	switch strings.TrimSpace(value) {
	case "moment", "user", "group":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

var ErrInvalidFeedRequest = errors.New("feed: invalid request")
var ErrInvalidCursor = errors.New("feed: invalid cursor")
var ErrDismissTargetNotFound = errors.New("feed: dismiss target not found")
