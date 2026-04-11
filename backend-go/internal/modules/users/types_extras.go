package users

import (
	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
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

type RelativeLocation struct {
	Point          GeoPoint `json:"point"`
	LocationName   string   `json:"location_name"`
	Distance       int      `json:"distance,omitempty"`
	BearingDegrees int      `json:"bearing_degrees,omitempty"`
}

type UserInfo struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type SocialContext struct {
	MutualFriendsCount   int        `json:"mutual_friends_count"`
	MutualFriendsPreview []UserInfo `json:"mutual_friends_preview,omitempty"`
	CommonGroupsCount    *int       `json:"common_groups_count,omitempty"`
	CommonTags           []string   `json:"common_tags,omitempty"`
}

type UserItem struct {
	User           UserInfo           `json:"user"`
	Bio            string             `json:"bio,omitempty"`
	Tags           []string           `json:"tags,omitempty"`
	PresenceStatus UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline     string             `json:"last_online,omitempty"`
	Location       *RelativeLocation  `json:"location,omitempty"`
	Stats          *UserStats         `json:"stats,omitempty"`
	Relation       *Relationship      `json:"relation,omitempty"`
	SocialContext  *SocialContext     `json:"social_context,omitempty"`
}

type UserOpenTheDoorRequest struct {
	UserID string `json:"user_id"`
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
	NextCursor string     `json:"next_cursor,omitempty"`
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

type RequestEmailChangeRequest struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

type ConfirmEmailChangeRequest struct {
	NewEmail         string `json:"new_email"`
	VerificationCode string `json:"verification_code"`
}

type RequestPhoneChangeRequest struct {
	NewPhone string `json:"new_phone"`
	Password string `json:"password"`
}

type ConfirmPhoneChangeRequest struct {
	NewPhone         string `json:"new_phone"`
	VerificationCode string `json:"verification_code"`
}

type VerificationSent struct {
	ExpiresAt string `json:"expires_at"`
}

type DeleteAccountRequest struct {
	Confirmation string `json:"confirmation"`
}

type FollowRequest struct {
	UserID         string       `json:"user_id"`
	Source         FollowSource `json:"source,omitempty"`
	SourceMomentID string       `json:"source_moment_id,omitempty"`
}

type ToggleUserRequest struct {
	UserID string `json:"user_id"`
}

type BlockRequest struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason,omitempty"`
}

type SetRemarkRequest struct {
	UserID string `json:"user_id"`
	Remark string `json:"remark"`
}

type ListFriendsRequest struct {
	UserID     string            `json:"user_id,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type FriendItem struct {
	User           UserInfo            `json:"user"`
	PresenceStatus *UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline     *string             `json:"last_online,omitempty"`
	Remark         string              `json:"remark,omitempty"`
	FollowedAt     string              `json:"followed_at"`
	FollowedBackAt string              `json:"followed_back_at"`
}

type PaginatedFriends struct {
	Items      []FriendItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type ListFollowingRequest struct {
	UserID     string            `json:"user_id,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type FollowingItem struct {
	User           UserInfo            `json:"user"`
	FollowedAt     string              `json:"followed_at"`
	IsMutual       bool                `json:"is_mutual"`
	PresenceStatus *UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline     *string             `json:"last_online,omitempty"`
}

type PaginatedFollowing struct {
	Items      []FollowingItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type ListFollowersRequest struct {
	UserID     string            `json:"user_id,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type FollowerItem struct {
	User            UserInfo            `json:"user"`
	FollowedAt      string              `json:"followed_at"`
	IsFollowingBack bool                `json:"is_following_back"`
	PresenceStatus  *UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline      *string             `json:"last_online,omitempty"`
}

type PaginatedFollowers struct {
	Items      []FollowerItem `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type UserGroupsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type UserMomentsRequest struct {
	UserID     string            `json:"user_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type MyBlockedRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type BlockedUserItem struct {
	User      UserInfo `json:"user"`
	BlockedAt string   `json:"blocked_at"`
	Reason    string   `json:"reason,omitempty"`
}

type PaginatedBlockedUsers struct {
	Items      []BlockedUserItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type UserListGroupsResponse = grouptypes.PaginatedGroups

type UserListMomentsResponse = momenttypes.PaginatedMoments
