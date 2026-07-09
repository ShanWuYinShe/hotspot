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

type AnnouncementSeverity string

const (
	AnnouncementSeverityInfo    AnnouncementSeverity = "info"
	AnnouncementSeverityWarning AnnouncementSeverity = "warning"
	AnnouncementSeverityError   AnnouncementSeverity = "error"
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
	AnnouncementID string               `json:"announcement_id"`
	Title          string               `json:"title"`
	Content        string               `json:"content"`
	Severity       AnnouncementSeverity `json:"severity"`
	ActionURL      string               `json:"action_url,omitempty"`
	CreatedAt      string               `json:"created_at"`
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
	MaxGroupMembers            int `json:"max_group_members"`
	MaxMomentTags              int `json:"max_moment_tags"`
	MaxMomentImages            int `json:"max_moment_images"`
	MomentDescriptionMaxLength int `json:"moment_description_max_length"`
	CommentMaxLength           int `json:"comment_max_length"`
	SearchRadiusMax            int `json:"search_radius_max"`
}

type ContactInfo struct {
	SupportEmail      string `json:"support_email"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
	OfficialWebsite   string `json:"official_website"`
}

type ClientDefaults struct {
	DefaultLocation    Location `json:"default_location"`
	DefaultUserAvatar  string   `json:"default_user_avatar"`
	DefaultGroupAvatar string   `json:"default_group_avatar"`
}

type ClientConfig struct {
	Features FeatureSwitches `json:"features"`
	Limits   AppLimits       `json:"limits"`
	Contact  ContactInfo     `json:"contact"`
	Defaults ClientDefaults  `json:"defaults"`
}
