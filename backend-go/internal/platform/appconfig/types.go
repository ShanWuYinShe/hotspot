package appconfig

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Location struct {
	Point        GeoPoint `json:"point"`
	LocationName string   `json:"location_name"`
}

type GroupSettings struct {
	JoinMethod        string   `json:"join_method"`
	RequiresApproval  bool     `json:"requires_approval"`
	HasPassword       bool     `json:"has_password"`
	IsSearchable      bool     `json:"is_searchable"`
	MembersSearchable bool     `json:"members_searchable"`
	MaxMembers        int      `json:"max_members"`
	Location          Location `json:"location"`
}

type Features struct {
	EnablePublicRegistration bool `json:"enable_public_registration"`
	EnableGuestRegister      bool `json:"enable_guest_register"`
	EnableMoments            bool `json:"enable_moments"`
	EnableNearbyPeople       bool `json:"enable_nearby_people"`
}

type Limits struct {
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

// AuthConfig 认证安全运行参数（审查遗留项：验证码 TTL 与登录锁定阈值此前是
// auth 模块代码常量，调参需重新发布）。零值字段由 Effective 回退默认值，
// 存量配置无 auth section 时行为与迁移前完全一致。
type AuthConfig struct {
	// VerificationCodeTTLMinutes 注册/重置密码验证码有效期（分钟）。
	VerificationCodeTTLMinutes int `json:"verification_code_ttl_minutes"`
	// LoginLockoutThreshold 连续登录失败达到该次数即锁定账号。
	LoginLockoutThreshold int `json:"login_lockout_threshold"`
	// LoginLockoutMinutes 锁定时长（分钟）；同时作为失败计数衰减窗口
	//（与迁移前「衰减窗口 = 锁期」的语义一致）。
	LoginLockoutMinutes int `json:"login_lockout_minutes"`
}

const (
	DefaultVerificationCodeTTLMinutes = 10
	DefaultLoginLockoutThreshold      = 5
	DefaultLoginLockoutMinutes        = 15
)

// Effective 返回补齐默认值后的认证参数。
func (c AuthConfig) Effective() AuthConfig {
	if c.VerificationCodeTTLMinutes <= 0 {
		c.VerificationCodeTTLMinutes = DefaultVerificationCodeTTLMinutes
	}
	if c.LoginLockoutThreshold <= 0 {
		c.LoginLockoutThreshold = DefaultLoginLockoutThreshold
	}
	if c.LoginLockoutMinutes <= 0 {
		c.LoginLockoutMinutes = DefaultLoginLockoutMinutes
	}
	return c
}

type Contact struct {
	SupportEmail      string `json:"support_email"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
	OfficialWebsite   string `json:"official_website_url"`
}

type UploadConfig struct {
	ImageMaxSize int `json:"image_max_size"`
	VideoMaxSize int `json:"video_max_size"`
	AudioMaxSize int `json:"audio_max_size"`
	FileMaxSize  int `json:"file_max_size"`
	// AvatarMaxSize / ChatFileMaxSize 是头像类与聊天文件类的独立上限
	//（审查 P2-23：media 模块此前把 5MB/100MB 硬编码在代码里，与其余四类
	// 走 AppConfig 形成「同类参数两种来源」）。新增字段而非复用 image/file：
	// avatar 历史上限 5MB（低于图片 10MB）、chat file 历史上限 100MB（高于
	// file 50MB），复用会改变既有行为；存量配置无该字段 → 零值 → media
	// 回退到同值常量，行为不变。
	AvatarMaxSize     int      `json:"avatar_max_size"`
	ChatFileMaxSize   int      `json:"chat_file_max_size"`
	AllowedImageTypes []string `json:"allowed_image_types"`
	AllowedVideoTypes []string `json:"allowed_video_types"`
}

type Defaults struct {
	DefaultLocation      Location      `json:"default_location"`
	DefaultUserAvatar    string        `json:"default_user_avatar_url"`
	DefaultGroupAvatar   string        `json:"default_group_avatar_url"`
	DefaultGroupSettings GroupSettings `json:"default_group_settings"`
}

type RateLimitConfig struct {
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

type ResponseDefaults struct {
	SearchSuggestCount       int `json:"search_suggest_count"`
	HotMomentsCount          int `json:"hot_moments_count"`
	ChatRecentMessagesCount  int `json:"chat_recent_messages_count"`
	GroupAnnouncementsCount  int `json:"group_announcements_count"`
	SystemAnnouncementsCount int `json:"system_announcements_count"`
	CommentTopRepliesCount   int `json:"comment_top_replies_count"`
	UserProfileMomentsCount  int `json:"user_profile_moments_count"`
}

type AppConfig struct {
	Features         Features         `json:"features"`
	Limits           Limits           `json:"limits"`
	Contact          Contact          `json:"contact"`
	Upload           UploadConfig     `json:"upload"`
	Defaults         Defaults         `json:"defaults"`
	RateLimits       RateLimitConfig  `json:"rate_limits"`
	ResponseDefaults ResponseDefaults `json:"response_defaults"`
	Auth             AuthConfig       `json:"auth"`
	UpdatedAt        string           `json:"updated_at,omitempty"`
	UpdatedByUserID  string           `json:"updated_by_user_id,omitempty"`
}
