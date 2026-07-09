package users

import (
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/types"
)

type UserType string

type UserPresenceStatus string

type Gender string

const (
	UserTypeTemp   UserType = "temp"
	UserTypeNormal UserType = "normal"

	UserPresenceOnline  UserPresenceStatus = "online"
	UserPresenceOffline UserPresenceStatus = "offline"
)

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Location struct {
	Point        GeoPoint `json:"point"`
	LocationName string   `json:"location_name"`
}

type UserStats struct {
	MomentsCount   int `json:"moments_count"`
	FriendsCount   int `json:"friends_count"`
	FollowersCount int `json:"followers_count"`
	FollowingCount int `json:"following_count"`
	GroupsCount    int `json:"groups_count"`
}

type MyProfile struct {
	UserID         string             `json:"user_id"`
	Nickname       string             `json:"nickname"`
	AvatarURL      string             `json:"avatar_url,omitempty"`
	UserType       UserType           `json:"user_type"`
	Bio            string             `json:"bio,omitempty"`
	Tags           []string           `json:"tags,omitempty"`
	Gender         Gender             `json:"gender,omitempty"`
	Email          string             `json:"email,omitempty"`
	Phone          string             `json:"phone,omitempty"`
	Location       Location           `json:"location"`
	PresenceStatus UserPresenceStatus `json:"presence_status"`
	LastOnline     string             `json:"last_online"`
	CreatedAt      string             `json:"created_at"`
	Stats          UserStats          `json:"stats"`
}

type PrivateChatPermission string

const (
	PrivateChatEveryone PrivateChatPermission = "everyone"
	PrivateChatFriends  PrivateChatPermission = "friends"
	PrivateChatNobody   PrivateChatPermission = "nobody"
)

type UserSettings struct {
	PrivacyLevel        string                `json:"privacy_level"`
	LocationVisible     bool                  `json:"location_visible"`
	PresenceVisible     bool                  `json:"presence_visible"`
	SocialListsVisible  bool                  `json:"social_lists_visible"`
	GroupsVisible       bool                  `json:"groups_visible"`
	LikedMomentsVisible bool                  `json:"liked_moments_visible"`
	StatsVisible        bool                  `json:"stats_visible"`
	AllowFollow         bool                  `json:"allow_follow"`
	AllowPrivateChat    PrivateChatPermission `json:"allow_private_chat"`
	Searchable          bool                  `json:"searchable"`
	PhoneSearchable     bool                  `json:"phone_searchable"`
	NicknameSearchable  bool                  `json:"nickname_searchable"`
	LocationSearchable  bool                  `json:"location_searchable"`
}

type UpdateProfileRequest struct {
	Nickname     *string   `json:"nickname,omitempty"`
	AvatarFileID *string   `json:"avatar_file_id,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	Gender       *Gender   `json:"gender,omitempty"`
}

type PartialUserSettings struct {
	PrivacyLevel        *string                `json:"privacy_level,omitempty"`
	LocationVisible     *bool                  `json:"location_visible,omitempty"`
	PresenceVisible     *bool                  `json:"presence_visible,omitempty"`
	SocialListsVisible  *bool                  `json:"social_lists_visible,omitempty"`
	GroupsVisible       *bool                  `json:"groups_visible,omitempty"`
	LikedMomentsVisible *bool                  `json:"liked_moments_visible,omitempty"`
	StatsVisible        *bool                  `json:"stats_visible,omitempty"`
	AllowFollow         *bool                  `json:"allow_follow,omitempty"`
	AllowPrivateChat    *PrivateChatPermission `json:"allow_private_chat,omitempty"`
	Searchable          *bool                  `json:"searchable,omitempty"`
	PhoneSearchable     *bool                  `json:"phone_searchable,omitempty"`
	NicknameSearchable  *bool                  `json:"nickname_searchable,omitempty"`
	LocationSearchable  *bool                  `json:"location_searchable,omitempty"`
}

type UpdateSettingsRequest struct {
	Settings PartialUserSettings `json:"settings"`
}

type PaginationParams = types.PaginationParams

type UserLikedMomentsRequest struct {
	UserID     string                        `json:"user_id"`
	Pagination *momenttypes.PaginationParams `json:"pagination,omitempty"`
}
