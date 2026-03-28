package users

import momenttypes "hotspot/backend-go/internal/modules/moments"

type Relationship string

const (
	RelationshipNone      Relationship = "none"
	RelationshipFollowing Relationship = "following"
	RelationshipFollower  Relationship = "follower"
	RelationshipFriend    Relationship = "friend"
	RelationshipBlocked   Relationship = "blocked"
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

type UserItem struct {
	User           UserInfo           `json:"user"`
	Bio            string             `json:"bio,omitempty"`
	Tags           []string           `json:"tags,omitempty"`
	PresenceStatus UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline     string             `json:"last_online,omitempty"`
	Location       *RelativeLocation  `json:"location,omitempty"`
	Stats          *UserStats         `json:"stats,omitempty"`
	Relation       Relationship       `json:"relation,omitempty"`
}

type UserOpenTheDoorRequest struct {
	UserID string `json:"user_id"`
}

type SearchUsersRequest struct {
	Keyword string `json:"keyword,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Radius  int    `json:"radius,omitempty"`
}

type PaginatedUsers struct {
	Items      []UserItem `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
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

type ToggleUserRequest struct {
	UserID string `json:"user_id"`
}

type SetRemarkRequest struct {
	UserID string `json:"user_id"`
	Remark string `json:"remark"`
}

type BlockedUserItem struct {
	User   UserInfo `json:"user"`
	Remark string   `json:"remark,omitempty"`
}

type PaginatedBlockedUsers struct {
	Items      []BlockedUserItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type UserLikedMomentsRequest struct {
	UserID     string                     `json:"user_id"`
	Pagination *momenttypes.PaginationParams `json:"pagination,omitempty"`
}
