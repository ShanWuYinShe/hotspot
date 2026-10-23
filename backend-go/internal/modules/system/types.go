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

type AnnounceSeverity string

const (
	AnnounceLevelInfo    AnnounceSeverity = "info"
	AnnounceLevelWarning AnnounceSeverity = "warning"
	AnnounceLevelError   AnnounceSeverity = "error"
)

type GeoPoint = types.GeoPoint

type Location struct {
	Point        GeoPoint `json:"point"`
	LocationName string   `json:"location_name"`
}

type ApiVersionMeta struct {
	Version            string           `json:"version"`
	Status             ApiVersionStatus `json:"status"`
	DeprecatedAt       *string          `json:"deprecated_at"`
	SunsetAt           *string          `json:"sunset_at"`
	Message            *string          `json:"message"`
	RecommendedVersion *string          `json:"recommended_version"`
}

type HealthStatus struct {
	Status    SystemHealthStatus `json:"status"`
	Timestamp string             `json:"timestamp"`
	// 版本指纹（构建期经 -ldflags 注入，spec api_spec_types.md HealthStatus）：
	// 线上排障时确认当前运行产物；本地裸构建时为 version=dev、其余为空串。
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

type SystemAnnouncementItem struct {
	AnnouncementID string           `json:"announcement_id"`
	Title          string           `json:"title"`
	Content        string           `json:"content"`
	Severity       AnnounceSeverity `json:"severity"`
	ActionURL      *string          `json:"action_url"`
	CreatedAt      string           `json:"created_at"`
}

type SystemAnnouncementList struct {
	Items []SystemAnnouncementItem `json:"items"`
}

type FeatureSwitches struct {
	EnablePublicRegistration bool `json:"enable_public_registration"`
	EnableGuestRegister      bool `json:"enable_guest_register"`
	EnableMoments            bool `json:"enable_moments"`
	EnableNearbyPeople       bool `json:"enable_nearby_people"`
}

type AppLimits struct {
	MaxGroupMembers            int      `json:"max_group_members"`
	MaxMomentTags              int      `json:"max_moment_tags"`
	MaxMomentImages            int      `json:"max_moment_images"`
	MaxChatFiles               int      `json:"max_chat_files"`
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
	BootstrapVersion string          `json:"bootstrap_version"`
	Meta             *ApiVersionMeta `json:"meta"`           // 恒定 nullable；缓存命中时整体省略（handler 用精简类型）
	ClientConfig     *ClientConfig   `json:"client_config"`  // 恒定 nullable；缓存命中时整体省略
	Profile          any             `json:"profile"`        // 恒定 nullable：匿名时为 null
	UnreadSummary    *UnreadSummary  `json:"unread_summary"` // 恒定 nullable：匿名时为 null
}

// BootstrapCacheHitData 缓存命中时的精简响应：meta/client_config 为公共
// 数据，命中时整体省略（客户端用缓存）；profile/unread_summary 为用户
// 私有数据、无缓存语义，命中时仍恒返回最新值（spec api_spec_system.md
// 缓存契约——spec 类型定义处的「命中时同样省略」注释与该 prose 冲突，
// 以更明确且带设计论证的 prose 为准：否则登录态客户端命中缓存后将拿
// 不到任何私有数据，聚合端点失去意义，审查 P1-2）。
type BootstrapCacheHitData struct {
	BootstrapVersion string         `json:"bootstrap_version"`
	Profile          any            `json:"profile"`        // 恒定 nullable：匿名时为 null
	UnreadSummary    *UnreadSummary `json:"unread_summary"` // 恒定 nullable：匿名时为 null
}

type UnreadSummary struct {
	NotifyTotal            int            `json:"notify_total"`
	NotifyByType           map[string]int `json:"notify_by_type"`
	PrivateChatUnreadCount int            `json:"private_chat_unread_count"`
	GroupChatUnreadCount   int            `json:"group_chat_unread_count"`
	TotalBadge             int            `json:"total_badge"`
}
