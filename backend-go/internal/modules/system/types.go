package system

import "hotspot/backend-go/internal/platform/types"

type ApiVersionStatus string

const (
	ApiVersionStatusActive     ApiVersionStatus = "active"
	ApiVersionStatusDeprecated ApiVersionStatus = "deprecated"
	ApiVersionStatusSunset     ApiVersionStatus = "sunset"
)

type SystemHealthStatus string

const (
	SystemHealthHealthy   SystemHealthStatus = "healthy"
	SystemHealthDegraded  SystemHealthStatus = "degraded"
	SystemHealthUnhealthy SystemHealthStatus = "unhealthy"
)

type AnnounceLevel string

const (
	AnnounceLevelInfo    AnnounceLevel = "info"
	AnnounceLevelWarning AnnounceLevel = "warning"
	AnnounceLevelError   AnnounceLevel = "error"
)

type GeoPoint = types.GeoPoint

type Location struct {
	Point        GeoPoint `json:"point"`
	LocationName string   `json:"location_name"`
}

type ApiVersionMeta struct {
	Version            string           `json:"version"`
	Status             ApiVersionStatus `json:"status"`
	DeprecatedAt       string           `json:"deprecated_at,omitempty"`
	SunsetAt           string           `json:"sunset_at,omitempty"`
	Message            string           `json:"message,omitempty"`
	RecommendedVersion string           `json:"recommended_version,omitempty"`
}

type HealthStatus struct {
	Status    SystemHealthStatus `json:"status"`
	Timestamp string             `json:"timestamp"`
}

type SystemAnnouncementItem struct {
	AnnouncementID string        `json:"announcement_id"`
	Title          string        `json:"title"`
	Content        string        `json:"content"`
	Severity       AnnounceLevel `json:"severity"`
	ActionURL      string        `json:"action_url,omitempty"`
	CreatedAt      string        `json:"created_at"`
}

type SystemAnnouncementList struct {
	Items []SystemAnnouncementItem `json:"items"`
}

type FeatureSwitches struct {
	EnablePublicRegistration bool `json:"enable_public_registration"`
	EnableGuestLogin         bool `json:"enable_guest_login"`
	EnableMoments            bool `json:"enable_moments"`
	EnableNearbyPeople       bool `json:"enable_nearby_people"`
}

type AppLimits struct {
	MaxGroupMembers            int      `json:"max_group_members"`
	MaxMomentTags              int      `json:"max_moment_tags"`
	MaxMomentImages            int      `json:"max_moment_images"`
	MomentDescriptionMaxLength int      `json:"moment_description_max_length"`
	CommentMaxLength           int      `json:"comment_max_length"`
	SearchRadiusMax            int      `json:"search_radius_max"`
	MessageRecallWindowSeconds int      `json:"message_recall_window_seconds"`
	StrangerMessageDailyQuota  int      `json:"stranger_message_daily_quota"`
	ReactionEmojis             []string `json:"reaction_emojis"`
	GroupInviteTTLHours        int      `json:"group_invite_ttl_hours"`
	GroupApplicationTTLHours   int      `json:"group_application_ttl_hours"`
}

type ContactInfo struct {
	SupportEmail      string `json:"support_email"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
	OfficialWebsite   string `json:"official_website_url"`
}

type ClientDefaults struct {
	DefaultLocation    Location `json:"default_location"`
	DefaultUserAvatar  string   `json:"default_user_avatar_url"`
	DefaultGroupAvatar string   `json:"default_group_avatar_url"`
}

type ClientConfig struct {
	Features FeatureSwitches `json:"features"`
	Limits   AppLimits       `json:"limits"`
	Contact  ContactInfo     `json:"contact"`
	Defaults ClientDefaults  `json:"defaults"`
}

type BootstrapRequest struct {
	ClientType string  `json:"client_type"`
	DeviceID   *string `json:"device_id,omitempty"`
}

type BootstrapData struct {
	BootstrapVersion string         `json:"bootstrap_version"`
	Meta             *ApiVersionMeta `json:"meta"`         // 恒定 nullable；缓存命中时整体省略（handler 用精简类型）
	ClientConfig     *ClientConfig   `json:"client_config"` // 恒定 nullable；缓存命中时整体省略
	Profile          any             `json:"profile"`       // 恒定 nullable：匿名时为 null
	UnreadSummary    *UnreadSummary  `json:"unread_summary"` // 恒定 nullable：匿名时为 null
}

// BootstrapCacheHitData 缓存命中时的精简响应（仅 bootstrap_version，
// meta/client_config/profile/unread_summary 由客户端用缓存）。
type BootstrapCacheHitData struct {
	BootstrapVersion string `json:"bootstrap_version"`
}

type UnreadSummary struct {
	NotifyTotal            int            `json:"notify_total"`
	NotifyByType           map[string]int `json:"notify_by_type"`
	PrivateChatUnreadCount int            `json:"private_chat_unread_count"`
	GroupChatUnreadCount   int            `json:"group_chat_unread_count"`
	TotalBadge             int            `json:"total_badge"`
}
