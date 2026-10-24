package admin

import (
	groupsmodule "hotspot/backend-go/internal/modules/groups"
)

type AdminUploadConfig struct {
	ImageMaxSize      int      `json:"image_max_size"`
	VideoMaxSize      int      `json:"video_max_size"`
	AudioMaxSize      int      `json:"audio_max_size"`
	FileMaxSize       int      `json:"file_max_size"`
	AvatarMaxSize     int      `json:"avatar_max_size"`
	ChatFileMaxSize   int      `json:"chat_file_max_size"`
	AllowedImageTypes []string `json:"allowed_image_types"`
	AllowedVideoTypes []string `json:"allowed_video_types"`
}

type AdminGroupSettings struct {
	JoinMethod        groupsmodule.GroupJoinMethod `json:"join_method"`
	RequiresApproval  bool                         `json:"requires_approval"`
	HasPassword       bool                         `json:"has_password"`
	IsSearchable      bool                         `json:"is_searchable"`
	MembersSearchable bool                         `json:"members_searchable"`
	MaxMembers        int                          `json:"max_members"`
	Location          Location                     `json:"location"`
}

type ContentDefaults struct {
	DefaultLocation      Location           `json:"default_location"`
	DefaultGroupAvatar   string             `json:"default_group_avatar_url"`
	DefaultUserAvatar    string             `json:"default_user_avatar_url"`
	DefaultGroupSettings AdminGroupSettings `json:"default_group_settings"`
}

type AdminRateLimitConfig struct {
	GlobalPerHour            int `json:"global_per_hour"`
	UserPerHour              int `json:"user_per_hour"`
	AuthRegisterPer10Minutes int `json:"auth_register_per_10minutes"`
	AuthSigninPer10Minutes   int `json:"auth_signin_per_10minutes"`
	AuthGuestPer10Minutes    int `json:"auth_guest_per_10minutes"`
	AuthRefreshPer10Minutes  int `json:"auth_refresh_per_10minutes"`
	AuthResetPer10Minutes    int `json:"auth_reset_per_10minutes"`
	SearchPerMinute          int `json:"search_per_minute"`
	WritePerMinute           int `json:"write_per_minute"`
	MomentsCreatePerMinute   int `json:"moments_create_per_minute"`
	CommentsCreatePerMinute  int `json:"comments_create_per_minute"`
	ReportsCreatePerHour     int `json:"reports_create_per_hour"`
	UploadPresignPerHour     int `json:"upload_presign_per_hour"`
	WSSubscribePerSecond     int `json:"ws_subscribe_per_second"`
	WSHandshakePerMinute     int `json:"ws_handshake_per_minute"`
	AdminReadPerMinute       int `json:"admin_read_per_minute"`
	AdminWritePerMinute      int `json:"admin_write_per_minute"`
}

type AdminResponseDefaults struct {
	SearchSuggestCount       int `json:"search_suggest_count"`
	HotMomentsCount          int `json:"hot_moments_count"`
	ChatRecentMessagesCount  int `json:"chat_recent_messages_count"`
	GroupAnnouncementsCount  int `json:"group_announcements_count"`
	SystemAnnouncementsCount int `json:"system_announcements_count"`
	CommentTopRepliesCount   int `json:"comment_top_replies_count"`
	UserProfileMomentsCount  int `json:"user_profile_moments_count"`
}

type AdminAuthConfig struct {
	VerificationCodeTTLMinutes int `json:"verification_code_ttl_minutes"`
	LoginLockoutThreshold      int `json:"login_lockout_threshold"`
	LoginLockoutMinutes        int `json:"login_lockout_minutes"`
}

type AdminAppConfig struct {
	Features         FeatureSwitches       `json:"features"`
	Limits           AppLimits             `json:"limits"`
	Contact          ContactInfo           `json:"contact"`
	Upload           AdminUploadConfig     `json:"upload"`
	Defaults         ContentDefaults       `json:"defaults"`
	RateLimits       AdminRateLimitConfig  `json:"rate_limits"`
	ResponseDefaults AdminResponseDefaults `json:"response_defaults"`
	Auth             AdminAuthConfig       `json:"auth"`
	UpdatedAt        *string               `json:"updated_at"`
	UpdatedByUserID  *string               `json:"updated_by_user_id"`
}

type UpdateAppConfigRequest struct {
	Upload           *PartialUploadConfig     `json:"upload,omitempty"`
	Defaults         *UpdateDefaultsPayload   `json:"defaults,omitempty"`
	RateLimits       *PartialRateLimitConfig  `json:"rate_limits,omitempty"`
	ResponseDefaults *PartialResponseDefaults `json:"response_defaults,omitempty"`
	Auth             *PartialAuthConfig       `json:"auth,omitempty"`
	Features         *PartialFeatureSwitches  `json:"features,omitempty"`
	Limits           *PartialAppLimits        `json:"limits,omitempty"`
	Contact          *PartialContactInfo      `json:"contact,omitempty"`
}

type UpdateDefaultsPayload struct {
	DefaultLocation      *PartialLocation      `json:"default_location,omitempty"`
	DefaultGroupAvatar   *string               `json:"default_group_avatar_url,omitempty"`
	DefaultUserAvatar    *string               `json:"default_user_avatar_url,omitempty"`
	DefaultGroupSettings *PartialGroupSettings `json:"default_group_settings,omitempty"`
}

type PartialFeatureSwitches struct {
	EnablePublicRegistration *bool `json:"enable_public_registration,omitempty"`
	// spec api_spec_types.md FeatureSwitches：键为 enable_guest_register
	EnableGuestRegister *bool `json:"enable_guest_register,omitempty"`
	EnableMoments       *bool `json:"enable_moments,omitempty"`
	EnableNearbyPeople  *bool `json:"enable_nearby_people,omitempty"`
}

type PartialContactInfo struct {
	SupportEmail      *string `json:"support_email,omitempty"`
	PrivacyPolicyURL  *string `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL *string `json:"terms_of_service_url,omitempty"`
	OfficialWebsite   *string `json:"official_website_url,omitempty"`
}

type PartialUploadConfig struct {
	ImageMaxSize      *int     `json:"image_max_size,omitempty"`
	VideoMaxSize      *int     `json:"video_max_size,omitempty"`
	AudioMaxSize      *int     `json:"audio_max_size,omitempty"`
	FileMaxSize       *int     `json:"file_max_size,omitempty"`
	AvatarMaxSize     *int     `json:"avatar_max_size,omitempty"`
	ChatFileMaxSize   *int     `json:"chat_file_max_size,omitempty"`
	AllowedImageTypes []string `json:"allowed_image_types,omitempty"`
	AllowedVideoTypes []string `json:"allowed_video_types,omitempty"`
}

// PartialAuthConfig 认证安全参数的部分更新（spec api_spec_admin.md「应用配置子类型定义」）。
type PartialAuthConfig struct {
	VerificationCodeTTLMinutes *int `json:"verification_code_ttl_minutes,omitempty"`
	LoginLockoutThreshold      *int `json:"login_lockout_threshold,omitempty"`
	LoginLockoutMinutes        *int `json:"login_lockout_minutes,omitempty"`
}

type PartialLocation struct {
	Point        *GeoPoint `json:"point,omitempty"`
	LocationName *string   `json:"location_name,omitempty"`
}

type PartialGroupSettings struct {
	JoinMethod        *groupsmodule.GroupJoinMethod `json:"join_method,omitempty"`
	RequiresApproval  *bool                         `json:"requires_approval,omitempty"`
	HasPassword       *bool                         `json:"has_password,omitempty"`
	IsSearchable      *bool                         `json:"is_searchable,omitempty"`
	MembersSearchable *bool                         `json:"members_searchable,omitempty"`
	MaxMembers        *int                          `json:"max_members,omitempty"`
	Location          *PartialLocation              `json:"location,omitempty"`
}

type PartialRateLimitConfig struct {
	GlobalPerHour            *int `json:"global_per_hour,omitempty"`
	UserPerHour              *int `json:"user_per_hour,omitempty"`
	AuthRegisterPer10Minutes *int `json:"auth_register_per_10minutes,omitempty"`
	AuthSigninPer10Minutes   *int `json:"auth_signin_per_10minutes,omitempty"`
	AuthGuestPer10Minutes    *int `json:"auth_guest_per_10minutes,omitempty"`
	AuthRefreshPer10Minutes  *int `json:"auth_refresh_per_10minutes,omitempty"`
	AuthResetPer10Minutes    *int `json:"auth_reset_per_10minutes,omitempty"`
	SearchPerMinute          *int `json:"search_per_minute,omitempty"`
	WritePerMinute           *int `json:"write_per_minute,omitempty"`
	MomentsCreatePerMinute   *int `json:"moments_create_per_minute,omitempty"`
	CommentsCreatePerMinute  *int `json:"comments_create_per_minute,omitempty"`
	ReportsCreatePerHour     *int `json:"reports_create_per_hour,omitempty"`
	UploadPresignPerHour     *int `json:"upload_presign_per_hour,omitempty"`
	WSSubscribePerSecond     *int `json:"ws_subscribe_per_second,omitempty"`
	WSHandshakePerMinute     *int `json:"ws_handshake_per_minute,omitempty"`
	AdminReadPerMinute       *int `json:"admin_read_per_minute,omitempty"`
	AdminWritePerMinute      *int `json:"admin_write_per_minute,omitempty"`
}

type PartialResponseDefaults struct {
	SearchSuggestCount       *int `json:"search_suggest_count,omitempty"`
	HotMomentsCount          *int `json:"hot_moments_count,omitempty"`
	ChatRecentMessagesCount  *int `json:"chat_recent_messages_count,omitempty"`
	GroupAnnouncementsCount  *int `json:"group_announcements_count,omitempty"`
	SystemAnnouncementsCount *int `json:"system_announcements_count,omitempty"`
	CommentTopRepliesCount   *int `json:"comment_top_replies_count,omitempty"`
	UserProfileMomentsCount  *int `json:"user_profile_moments_count,omitempty"`
}

type PartialAppLimits struct {
	MaxGroupMembers            *int     `json:"max_group_members,omitempty"`
	MaxMomentTags              *int     `json:"max_moment_tags,omitempty"`
	MaxMomentImages            *int     `json:"max_moment_images,omitempty"`
	MaxChatFiles               *int     `json:"max_chat_files,omitempty"`
	MomentDescriptionMaxLength *int     `json:"moment_description_max_length,omitempty"`
	CommentMaxLength           *int     `json:"comment_max_length,omitempty"`
	SearchRadiusMax            *int     `json:"search_radius_max,omitempty"`
	MessageRecallWindowSeconds *int     `json:"message_recall_window_seconds,omitempty"`
	StrangerMessageDailyQuota  *int     `json:"stranger_message_daily_quota,omitempty"`
	ReactionEmojis             []string `json:"reaction_emojis,omitempty"`
	GroupInviteTTLHours        *int     `json:"group_invite_ttl_hours,omitempty"`
	GroupApplicationTTLHours   *int     `json:"group_application_ttl_hours,omitempty"`
}

type AdminTagItem struct {
	TagID      string `json:"tag_id"`
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
	CreatedAt  string `json:"created_at"`
	// IsBlocked：标签是否已被软屏蔽（spec api_spec_admin.md tags/search）
	IsBlocked bool `json:"is_blocked"`
}

type PaginatedAdminTags struct {
	Items      []AdminTagItem `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

type AdminTagSearchRequest struct {
	Keyword    string            `json:"keyword,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// AdminSetTagBanRequest 是两态统一端点 admin/tags/set_ban 的请求类型（spec
// api_spec_admin.md）：active=true 屏蔽 / false 解屏蔽；hard_delete 仅
// active=true 时有效（物理删除历史引用，不可逆）。
type AdminSetTagBanRequest struct {
	Name       string `json:"name"`
	Active     bool   `json:"active"`
	HardDelete bool   `json:"hard_delete,omitempty"`
}

type AdminAnnouncementItem struct {
	AnnouncementID string           `json:"announcement_id"`
	Title          string           `json:"title"`
	Content        string           `json:"content"`
	Severity       AnnounceSeverity `json:"severity"`
	ActionURL      *string          `json:"action_url"`
	CreatedAt      string           `json:"created_at"`
	CreatedBy      UserInfo         `json:"created_by"`
	SendPush       bool             `json:"send_push"`
	PushSentAt     *string          `json:"push_sent_at"`
	ViewCount      int              `json:"view_count"`
}

type PaginatedAdminAnnouncements struct {
	Items      []AdminAnnouncementItem `json:"items"`
	NextCursor *string                 `json:"next_cursor"`
}

type ListAdminAnnouncementsRequest struct {
	Severity   AnnounceSeverity  `json:"severity,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type CreateAnnouncementRequest struct {
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Severity  AnnounceSeverity `json:"severity"`
	ActionURL string           `json:"action_url,omitempty"`
	SendPush  bool             `json:"send_push,omitempty"`
}

type UpdateAnnouncementRequest struct {
	AnnouncementID string            `json:"announcement_id"`
	Title          *string           `json:"title,omitempty"`
	Content        *string           `json:"content,omitempty"`
	Severity       *AnnounceSeverity `json:"severity,omitempty"`
	ActionURL      *string           `json:"action_url,omitempty"`
}

type DeleteAnnouncementRequest struct {
	AnnouncementID string `json:"announcement_id"`
}
