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
	JoinMethod   string   `json:"join_method"`
	IsSearchable bool     `json:"is_searchable"`
	MaxMembers   int      `json:"max_members"`
	Location     Location `json:"location"`
}

type Features struct {
	EnablePublicRegistration bool `json:"enable_public_registration"`
	EnableGuestLogin         bool `json:"enable_guest_login"`
	EnableMoments            bool `json:"enable_moments"`
	EnableNearbyPeople       bool `json:"enable_nearby_people"`
}

type Limits struct {
	MaxGroupMembers            int `json:"max_group_members"`
	MaxMomentTags              int `json:"max_moment_tags"`
	MaxMomentImages            int `json:"max_moment_images"`
	MomentDescriptionMaxLength int `json:"moment_description_max_length"`
	CommentMaxLength           int `json:"comment_max_length"`
	SearchRadiusMax            int `json:"search_radius_max"`
}

type Contact struct {
	SupportEmail      string `json:"support_email"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
	OfficialWebsite   string `json:"official_website_url"`
}

type UploadConfig struct {
	ImageMaxSize      int      `json:"image_max_size"`
	VideoMaxSize      int      `json:"video_max_size"`
	AudioMaxSize      int      `json:"audio_max_size"`
	FileMaxSize       int      `json:"file_max_size"`
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
	AuthLoginPer10Minutes    int `json:"auth_login_per_10minutes"`
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
	AdminReadPerMinute       int `json:"admin_read_per_minute"`
	AdminWritePerMinute      int `json:"admin_write_per_minute"`
}

type ResponseDefaults struct {
	SearchSuggestCount       int `json:"search_suggest_count"`
	TrendingTagsCount        int `json:"trending_tags_count"`
	HotMomentsCount          int `json:"hot_moments_count"`
	ChatRecentMessagesCount  int `json:"chat_recent_messages_count"`
	GroupAnnouncementsCount  int `json:"group_announcements_count"`
	SystemAnnouncementsCount int `json:"system_announcements_count"`
	CommentTopRepliesCount   int `json:"comment_top_replies_count"`
	UserProfileMomentsCount  int `json:"user_profile_moments_count"`
	SearchAllCountPerType    int `json:"search_all_count_per_type"`
}

type AppConfig struct {
	Features         Features         `json:"features"`
	Limits           Limits           `json:"limits"`
	Contact          Contact          `json:"contact"`
	Upload           UploadConfig     `json:"upload"`
	Defaults         Defaults         `json:"defaults"`
	RateLimits       RateLimitConfig  `json:"rate_limits"`
	ResponseDefaults ResponseDefaults `json:"response_defaults"`
	UpdatedAt        string           `json:"updated_at,omitempty"`
	UpdatedByUserID  string           `json:"updated_by_user_id,omitempty"`
}
