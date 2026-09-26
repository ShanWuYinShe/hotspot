package users

import (
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/types"
)

type UserType string

type UserPresenceStatus string

type Gender string

// Gender 枚举（spec api_spec_user.md）：type Gender = 'male' | 'female' | 'other'。
const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

const (
	UserTypeTemp   UserType = "temp"
	UserTypeNormal UserType = "normal"

	UserPresenceOnline  UserPresenceStatus = "online"
	UserPresenceOffline UserPresenceStatus = "offline"
)

type GeoPoint = types.GeoPoint

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
	LastOnline     string             `json:"last_online_at"`
	CreatedAt      string             `json:"created_at"`
	Stats          UserStats          `json:"stats"`
	// IsDeleted 为内部标记（不序列化）：已注销用户（匿名化占位，见 api_spec_user.md）。
	IsDeleted bool `json:"-"`
}

// AccountData 是 users/account 与 users/upgrade 的响应形状（spec api_spec_user.md →
// AccountData）。与内部存储模型 MyProfile 分离，保证对外恒定 nullable。
type AccountData struct {
	UserID         string             `json:"user_id"`
	Nickname       string             `json:"nickname"`
	AvatarURL      *string            `json:"avatar_url"`
	UserType       UserType           `json:"user_type"`
	Bio            *string            `json:"bio"`
	Tags           []string           `json:"tags"`
	Gender         *Gender            `json:"gender"`
	Email          *string            `json:"email"`
	Phone          *string            `json:"phone"`
	Location       Location           `json:"location"`
	PresenceStatus UserPresenceStatus `json:"presence_status"`
	LastOnline     string             `json:"last_online_at"`
	CreatedAt      string             `json:"created_at"`
	Stats          UserStats          `json:"stats"`
}

func toAccountData(profile MyProfile) AccountData {
	tags := profile.Tags
	if tags == nil {
		tags = []string{}
	}
	var gender *Gender
	if profile.Gender != "" {
		g := profile.Gender
		gender = &g
	}
	return AccountData{
		UserID:         profile.UserID,
		Nickname:       profile.Nickname,
		AvatarURL:      types.StrPtr(profile.AvatarURL),
		UserType:       profile.UserType,
		Bio:            types.StrPtr(profile.Bio),
		Tags:           tags,
		Gender:         gender,
		Email:          types.StrPtr(profile.Email),
		Phone:          types.StrPtr(profile.Phone),
		Location:       profile.Location,
		PresenceStatus: profile.PresenceStatus,
		LastOnline:     profile.LastOnline,
		CreatedAt:      profile.CreatedAt,
		Stats:          profile.Stats,
	}
}

type ChatPerm string

const (
	ChatPermEveryone ChatPerm = "everyone"
	ChatPermFriends  ChatPerm = "friends"
	ChatPermNobody   ChatPerm = "nobody"
)

// Valid 报告该私聊权限是否为契约枚举值（spec api_spec_types.md ChatPerm）。
// 更新入口必须校验：未知值写入后，发送侧 switch 会 fall-through 放行，
// 导致隐私设置静默失效。
func (c ChatPerm) Valid() bool {
	switch c {
	case ChatPermEveryone, ChatPermFriends, ChatPermNobody:
		return true
	}
	return false
}

type UserSettings struct {
	PrivacyLevel       string `json:"privacy_level"`
	LocationVisible    bool   `json:"location_visible"`
	PresenceVisible    bool   `json:"presence_visible"`
	SocialListsVisible bool   `json:"social_lists_visible"`
	GroupsVisible      bool   `json:"groups_visible"`
	// ReactedMomentsVisible gates the list_reacted_moments endpoint. Renamed
	// from liked_moments_visible as part of the reaction migration
	// (see migrations/032_rename_liked_to_reacted_moments_visible.sql).
	ReactedMomentsVisible bool     `json:"reacted_moments_visible"`
	StatsVisible          bool     `json:"stats_visible"`
	AllowFollow           bool     `json:"allow_follow"`
	AllowPrivateChat      ChatPerm `json:"allow_private_chat"`
	Searchable            bool     `json:"searchable"`
	PhoneSearchable       bool     `json:"phone_searchable"`
	NicknameSearchable    bool     `json:"nickname_searchable"`
	LocationSearchable    bool     `json:"location_searchable"`
	NearbyMomentNotify    bool     `json:"nearby_moment_notify"`
}

type UpdateAccountRequest struct {
	Nickname     *string   `json:"nickname,omitempty"`
	AvatarFileID *string   `json:"avatar_file_id,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	Gender       *Gender   `json:"gender,omitempty"`
}

type PartialUserSettings struct {
	PrivacyLevel          *string   `json:"privacy_level,omitempty"`
	LocationVisible       *bool     `json:"location_visible,omitempty"`
	PresenceVisible       *bool     `json:"presence_visible,omitempty"`
	SocialListsVisible    *bool     `json:"social_lists_visible,omitempty"`
	GroupsVisible         *bool     `json:"groups_visible,omitempty"`
	ReactedMomentsVisible *bool     `json:"reacted_moments_visible,omitempty"`
	StatsVisible          *bool     `json:"stats_visible,omitempty"`
	AllowFollow           *bool     `json:"allow_follow,omitempty"`
	AllowPrivateChat      *ChatPerm `json:"allow_private_chat,omitempty"`
	Searchable            *bool     `json:"searchable,omitempty"`
	PhoneSearchable       *bool     `json:"phone_searchable,omitempty"`
	NicknameSearchable    *bool     `json:"nickname_searchable,omitempty"`
	LocationSearchable    *bool     `json:"location_searchable,omitempty"`
	NearbyMomentNotify    *bool     `json:"nearby_moment_notify,omitempty"`
}

type UpdateSettingsRequest struct {
	Settings PartialUserSettings `json:"settings"`
}

type PaginationParams = types.PaginationParams

type UserReactedMomentsRequest struct {
	UserID     string                        `json:"user_id"`
	Pagination *momenttypes.PaginationParams `json:"pagination,omitempty"`
}
