package admin

import (
	groupsmodule "hotspot/backend-go/internal/modules/groups"
	systemmodule "hotspot/backend-go/internal/modules/system"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"time"
)

type UserInfo = usersmodule.UserInfo

type GeoPoint = systemmodule.GeoPoint

type Location = systemmodule.Location

type FeatureSwitches = systemmodule.FeatureSwitches

type AppLimits = systemmodule.AppLimits

type ContactInfo = systemmodule.ContactInfo
type AnnouncementSeverity = systemmodule.AnnouncementSeverity

const (
	AnnouncementSeverityInfo    = systemmodule.AnnouncementSeverityInfo
	AnnouncementSeverityWarning = systemmodule.AnnouncementSeverityWarning
	AnnouncementSeverityError   = systemmodule.AnnouncementSeverityError
)

type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type AdminRoleBasic struct {
	RoleID string `json:"role_id"`
	Name   string `json:"name"`
}

type AdminPermissions struct {
	Permissions []string         `json:"permissions"`
	Roles       []AdminRoleBasic `json:"roles"`
}

type AdminPermissionItem struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type AdminPermissionList struct {
	Items []AdminPermissionItem `json:"items"`
}

type AdminRoleItem struct {
	RoleID      string   `json:"role_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

type AdminRoleList struct {
	Items []AdminRoleItem `json:"items"`
}

type AdminOperatorItem struct {
	User        UserInfo         `json:"user"`
	Permissions []string         `json:"permissions"`
	Roles       []AdminRoleBasic `json:"roles"`
}

type PaginatedAdminOperators struct {
	Items      []AdminOperatorItem `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type AdminRbacRoleCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type AdminRbacRoleUpdateRequest struct {
	RoleID           string  `json:"role_id"`
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ClearDescription bool    `json:"clear_description,omitempty"`
}

type AdminRbacRoleDeleteRequest struct {
	RoleID string `json:"role_id"`
}

type AdminRbacRoleSetPermissionsRequest struct {
	RoleID         string   `json:"role_id"`
	PermissionKeys []string `json:"permission_keys"`
}

type AdminRbacAdminsListRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminRbacGrantAdminRequest struct {
	UserID              string   `json:"user_id"`
	GrantAllPermissions bool     `json:"grant_all_permissions,omitempty"`
	RoleIDs             []string `json:"role_ids,omitempty"`
}

type AdminRbacRevokeAdminRequest struct {
	UserID string `json:"user_id"`
}

type UploadConfig struct {
	ImageMaxSize      int      `json:"image_max_size"`
	VideoMaxSize      int      `json:"video_max_size"`
	AudioMaxSize      int      `json:"audio_max_size"`
	FileMaxSize       int      `json:"file_max_size"`
	AllowedImageTypes []string `json:"allowed_image_types"`
	AllowedVideoTypes []string `json:"allowed_video_types"`
}

type GroupSettings struct {
	JoinMethod   groupsmodule.GroupJoinMethod `json:"join_method"`
	IsSearchable bool                         `json:"is_searchable"`
	MaxMembers   int                          `json:"max_members"`
	Location     Location                     `json:"location"`
}

type ContentDefaults struct {
	DefaultLocation      Location      `json:"default_location"`
	DefaultGroupAvatar   string        `json:"default_group_avatar"`
	DefaultUserAvatar    string        `json:"default_user_avatar"`
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
	SearchSuggestCount      int `json:"search_suggest_count"`
	TrendingTagsCount       int `json:"trending_tags_count"`
	HotMomentsCount         int `json:"hot_moments_count"`
	ChatRecentMessagesCount int `json:"chat_recent_messages_count"`
	GroupAnnouncementsCount int `json:"group_announcements_count"`
	SystemAnnouncementsCount int `json:"system_announcements_count"`
}

type AppConfig struct {
	Features         FeatureSwitches  `json:"features"`
	Limits           AppLimits        `json:"limits"`
	Contact          ContactInfo      `json:"contact"`
	Upload           UploadConfig     `json:"upload"`
	Defaults         ContentDefaults  `json:"defaults"`
	RateLimits       RateLimitConfig  `json:"rate_limits"`
	ResponseDefaults ResponseDefaults `json:"response_defaults"`
	UpdatedAt        string           `json:"updated_at,omitempty"`
	UpdatedByUserID  string           `json:"updated_by_user_id,omitempty"`
}

type AdminTagItem struct {
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
	Trend      string `json:"trend,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type PaginatedAdminTags struct {
	Items      []AdminTagItem `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type AdminTagSearchRequest struct {
	Keyword    string            `json:"keyword,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminDeleteTagRequest struct {
	Name string `json:"name"`
}

type AdminAnnouncementItem struct {
	AnnouncementID string               `json:"announcement_id"`
	Title          string               `json:"title"`
	Content        string               `json:"content"`
	Severity       AnnouncementSeverity `json:"severity"`
	ActionURL      string               `json:"action_url,omitempty"`
	CreatedAt      string               `json:"created_at"`
	CreatedBy      UserInfo             `json:"created_by"`
	SendPush       bool                 `json:"send_push"`
	PushSentAt     string               `json:"push_sent_at,omitempty"`
	ViewCount      int                  `json:"view_count"`
}

type PaginatedAdminAnnouncements struct {
	Items      []AdminAnnouncementItem `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type AdminAnnouncementSearchRequest struct {
	Severity   AnnouncementSeverity `json:"severity,omitempty"`
	Pagination *PaginationParams    `json:"pagination,omitempty"`
}

type CreateAnnouncementRequest struct {
	Title     string               `json:"title"`
	Content   string               `json:"content"`
	Severity  AnnouncementSeverity `json:"severity"`
	ActionURL string               `json:"action_url,omitempty"`
	SendPush  bool                 `json:"send_push,omitempty"`
}

type AnnouncementCreated struct {
	AnnouncementID string `json:"announcement_id"`
}

type UpdateAnnouncementRequest struct {
	AnnouncementID string                `json:"announcement_id"`
	Title          *string               `json:"title,omitempty"`
	Content        *string               `json:"content,omitempty"`
	Severity       *AnnouncementSeverity `json:"severity,omitempty"`
	ActionURL      *string               `json:"action_url,omitempty"`
}

type DeleteAnnouncementRequest struct {
	AnnouncementID string `json:"announcement_id"`
}

type AdminActor struct {
	UserID              string
	Permissions         []string
	Roles               []AdminRoleBasic
	GrantAllPermissions bool
}

func (a AdminActor) IsSuperAdmin() bool {
	return a.GrantAllPermissions || containsString(a.Permissions, wildcardPermission)
}

func (a AdminActor) HasPermission(key string) bool {
	return a.IsSuperAdmin() || containsString(a.Permissions, key)
}

func (a AdminActor) Response() AdminPermissions {
	permissions := append([]string(nil), a.Permissions...)
	roles := append([]AdminRoleBasic(nil), a.Roles...)
	return AdminPermissions{Permissions: permissions, Roles: roles}
}

type adminCursor struct {
	GrantedAt time.Time `json:"granted_at"`
	UserID    string    `json:"user_id"`
}

type adminListPage struct {
	Items []AdminOperatorItem
	Next  *adminCursor
}
