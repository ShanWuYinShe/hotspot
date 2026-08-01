package admin

import (
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	reportsmodule "hotspot/backend-go/internal/modules/reports"
	systemmodule "hotspot/backend-go/internal/modules/system"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/types"
)

type UserInfo = usersmodule.UserInfo

type GeoPoint = systemmodule.GeoPoint

type Location = systemmodule.Location

type FeatureSwitches = systemmodule.FeatureSwitches

type AppLimits = systemmodule.AppLimits

type ContactInfo = systemmodule.ContactInfo
type AnnounceLevel = systemmodule.AnnounceLevel

const (
	AnnounceLevelInfo    = systemmodule.AnnounceLevelInfo
	AnnounceLevelWarning = systemmodule.AnnounceLevelWarning
	AnnounceLevelError   = systemmodule.AnnounceLevelError
)

type PaginationParams = types.PaginationParams

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

type UserAccountStatus string

type ReportStatus string

type ReportTargetType string

type ReportAction string

const (
	UserAccountStatusNormal UserAccountStatus = "normal"
	UserAccountStatusBanned UserAccountStatus = "banned"

	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusEscalated ReportStatus = "escalated"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"

	ReportTargetTypeUser    ReportTargetType = "user"
	ReportTargetTypeMoment  ReportTargetType = "moment"
	ReportTargetTypeComment ReportTargetType = "comment"
	ReportTargetTypeMessage ReportTargetType = "message"
	ReportTargetTypeGroup   ReportTargetType = "group"

	ReportActionResolve  ReportAction = "resolve"
	ReportActionDismiss  ReportAction = "dismiss"
	ReportActionEscalate ReportAction = "escalate"
)

type AdminUserItem struct {
	User              usersmodule.UserInfo           `json:"user"`
	Bio               string                         `json:"bio,omitempty"`
	Tags              []string                       `json:"tags,omitempty"`
	PresenceStatus    usersmodule.UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline        string                         `json:"last_online,omitempty"`
	Location          *usersmodule.RelativeLocation  `json:"location,omitempty"`
	Stats             *usersmodule.UserStats         `json:"stats,omitempty"`
	Relation          usersmodule.Relationship       `json:"relation,omitempty"`
	Status            UserAccountStatus              `json:"status"`
	CreatedAt         string                         `json:"created_at"`
	LastActiveAt      string                         `json:"last_active_at,omitempty"`
	LoginID           string                         `json:"login_id"`
	RegisterIP        string                         `json:"register_ip,omitempty"`
	RegisterLocation  string                         `json:"register_location,omitempty"`
	LastLoginIP       string                         `json:"last_login_ip,omitempty"`
	LastLoginLocation string                         `json:"last_login_location,omitempty"`
	Email             string                         `json:"email,omitempty"`
	Phone             string                         `json:"phone,omitempty"`
}

type PaginatedAdminUsers struct {
	Items      []AdminUserItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type AdminUserSearchRequest struct {
	UserID     string               `json:"user_id,omitempty"`
	Nickname   string               `json:"nickname,omitempty"`
	LoginID    string               `json:"login_id,omitempty"`
	Status     UserAccountStatus    `json:"status,omitempty"`
	UserType   usersmodule.UserType `json:"user_type,omitempty"`
	Pagination *PaginationParams    `json:"pagination,omitempty"`
}

type AdminUserOpenTheDoorRequest struct {
	UserID string `json:"user_id"`
}

type AdminBanUserRequest struct {
	UserID       string `json:"user_id"`
	Reason       string `json:"reason,omitempty"`
	DurationDays *int   `json:"duration_days,omitempty"`
}

type AdminUnbanUserRequest struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason,omitempty"`
}

type AdminGroupItem struct {
	Group          groupsmodule.GroupInfo       `json:"group"`
	Description    string                       `json:"description,omitempty"`
	Tags           []string                     `json:"tags,omitempty"`
	HasPassword    bool                         `json:"has_password"`
	Location       groupsmodule.Location        `json:"location"`
	MyState        *groupsmodule.MyGroupState   `json:"my_state,omitempty"`
	JoinMethod     groupsmodule.GroupJoinMethod `json:"join_method,omitempty"`
	IsSearchable   bool                         `json:"is_searchable,omitempty"`
	MaxMembers     int                          `json:"max_members,omitempty"`
	CreatedAt      string                       `json:"created_at"`
	Owner          usersmodule.UserInfo         `json:"owner"`
	LastActivityAt string                       `json:"last_activity_at,omitempty"`
}

type PaginatedAdminGroups struct {
	Items      []AdminGroupItem `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type AdminGroupSearchRequest struct {
	GroupID    string            `json:"group_id,omitempty"`
	Name       string            `json:"name,omitempty"`
	OwnerID    string            `json:"owner_id,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminGroupOpenTheDoorRequest struct {
	GroupID string `json:"group_id"`
}

type AdminDismissGroupRequest struct {
	GroupID string `json:"group_id"`
	Reason  string `json:"reason,omitempty"`
}

type AdminReportItem struct {
	ReportID    string                `json:"report_id"`
	TargetType  ReportTargetType      `json:"target_type"`
	TargetID    string                `json:"target_id"`
	Reason      string                `json:"reason"`
	Description string                `json:"description,omitempty"`
	Status      ReportStatus          `json:"status"`
	CreatedAt   string                `json:"created_at"`
	HandledAt   string                `json:"handled_at,omitempty"`
	Reporter    usersmodule.UserInfo  `json:"reporter"`
	HandledBy   *usersmodule.UserInfo `json:"handled_by,omitempty"`
	HandleNote  string                `json:"handle_note,omitempty"`
}

type PaginatedAdminReports struct {
	Items      []AdminReportItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type AdminReportListRequest struct {
	Status     ReportStatus      `json:"status,omitempty"`
	TargetType ReportTargetType  `json:"target_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminReportHandleRequest struct {
	ReportID        string       `json:"report_id"`
	Action          ReportAction `json:"action"`
	Note            string       `json:"note,omitempty"`
	BanDurationDays *int         `json:"ban_duration_days,omitempty"`
}

type AuditAction string

type AuditTargetType string

type AuditDetails map[string]any

const (
	AuditActionAnnouncementCreate AuditAction = "announcement.create"
	AuditActionAnnouncementUpdate AuditAction = "announcement.update"
	AuditActionAnnouncementDelete AuditAction = "announcement.delete"
	AuditActionRoleCreate         AuditAction = "role.create"
	AuditActionRoleUpdate         AuditAction = "role.update"
	AuditActionRoleDelete         AuditAction = "role.delete"
	AuditActionRoleSetPermissions AuditAction = "role.set_permissions"
	AuditActionAdminGrant         AuditAction = "admin.grant"
	AuditActionAdminRevoke        AuditAction = "admin.revoke"
	AuditActionTagDelete          AuditAction = "tag.delete"
	AuditActionConfigUpdate       AuditAction = "config.update"
)

const (
	AuditTargetTypeAnnouncement AuditTargetType = "announcement"
	AuditTargetTypeRole         AuditTargetType = "role"
	AuditTargetTypeAdmin        AuditTargetType = "admin"
	AuditTargetTypeTag          AuditTargetType = "tag"
	AuditTargetTypeConfig       AuditTargetType = "config"
)

type AuditLogItem struct {
	LogID         string          `json:"log_id"`
	Action        AuditAction     `json:"action"`
	Operator      UserInfo        `json:"operator"`
	TargetType    AuditTargetType `json:"target_type"`
	TargetID      string          `json:"target_id"`
	TargetName    string          `json:"target_name,omitempty"`
	Details       AuditDetails    `json:"details,omitempty"`
	IPLocation    string          `json:"ip_location,omitempty"`
	IPAddress     *string         `json:"ip_address,omitempty"`
	DeviceSummary string          `json:"device_summary,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

type PaginatedAuditLogs struct {
	Items      []AuditLogItem `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type AuditLogListRequest struct {
	Action     AuditAction       `json:"action,omitempty"`
	OperatorID string            `json:"operator_id,omitempty"`
	TargetType AuditTargetType   `json:"target_type,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	StartTime  string            `json:"start_time,omitempty"`
	EndTime    string            `json:"end_time,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AuditLogOpenTheDoorRequest struct {
	LogID string `json:"log_id"`
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
	DefaultGroupAvatar   string        `json:"default_group_avatar_url"`
	DefaultUserAvatar    string        `json:"default_user_avatar_url"`
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
	HotMomentsCount          int `json:"hot_moments_count"`
	ChatRecentMessagesCount  int `json:"chat_recent_messages_count"`
	GroupAnnouncementsCount  int `json:"group_announcements_count"`
	SystemAnnouncementsCount int `json:"system_announcements_count"`
	CommentTopRepliesCount   int `json:"comment_top_replies_count"`
	UserProfileMomentsCount  int `json:"user_profile_moments_count"`
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

type UpdateAppConfigRequest struct {
	Upload           *PartialUploadConfig     `json:"upload,omitempty"`
	Defaults         *UpdateDefaultsPayload   `json:"defaults,omitempty"`
	RateLimits       *PartialRateLimitConfig  `json:"rate_limits,omitempty"`
	ResponseDefaults *PartialResponseDefaults `json:"response_defaults,omitempty"`
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
	EnableGuestLogin         *bool `json:"enable_guest_login,omitempty"`
	EnableMoments            *bool `json:"enable_moments,omitempty"`
	EnableNearbyPeople       *bool `json:"enable_nearby_people,omitempty"`
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
	AllowedImageTypes []string `json:"allowed_image_types,omitempty"`
	AllowedVideoTypes []string `json:"allowed_video_types,omitempty"`
}

type PartialLocation struct {
	Point        *GeoPoint `json:"point,omitempty"`
	LocationName *string   `json:"location_name,omitempty"`
}

type PartialGroupSettings struct {
	JoinMethod   *groupsmodule.GroupJoinMethod `json:"join_method,omitempty"`
	IsSearchable *bool                         `json:"is_searchable,omitempty"`
	MaxMembers   *int                          `json:"max_members,omitempty"`
	Location     *PartialLocation              `json:"location,omitempty"`
}

type PartialRateLimitConfig struct {
	GlobalPerHour            *int `json:"global_per_hour,omitempty"`
	UserPerHour              *int `json:"user_per_hour,omitempty"`
	AuthRegisterPer10Minutes *int `json:"auth_register_per_10minutes,omitempty"`
	AuthLoginPer10Minutes    *int `json:"auth_login_per_10minutes,omitempty"`
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
	Name       string `json:"name"`
	HardDelete bool   `json:"hard_delete,omitempty"`
}

type AdminAnnouncementItem struct {
	AnnouncementID string        `json:"announcement_id"`
	Title          string        `json:"title"`
	Content        string        `json:"content"`
	Severity       AnnounceLevel `json:"severity"`
	ActionURL      string        `json:"action_url,omitempty"`
	CreatedAt      string        `json:"created_at"`
	CreatedBy      UserInfo      `json:"created_by"`
	SendPush       bool          `json:"send_push"`
	PushSentAt     string        `json:"push_sent_at,omitempty"`
	ViewCount      int           `json:"view_count"`
}

type PaginatedAdminAnnouncements struct {
	Items      []AdminAnnouncementItem `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type AdminAnnouncementSearchRequest struct {
	Severity   AnnounceLevel     `json:"severity,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type CreateAnnouncementRequest struct {
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Severity  AnnounceLevel `json:"severity"`
	ActionURL string        `json:"action_url,omitempty"`
	SendPush  bool          `json:"send_push,omitempty"`
}

type AnnouncementCreated struct {
	AnnouncementID string `json:"announcement_id"`
}

type UpdateAnnouncementRequest struct {
	AnnouncementID string         `json:"announcement_id"`
	Title          *string        `json:"title,omitempty"`
	Content        *string        `json:"content,omitempty"`
	Severity       *AnnounceLevel `json:"severity,omitempty"`
	ActionURL      *string        `json:"action_url,omitempty"`
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

type auditCursor struct {
	CreatedAt time.Time `json:"created_at"`
	LogID     string    `json:"log_id"`
}

type AuditLogFilter struct {
	Action     AuditAction
	OperatorID string
	TargetType AuditTargetType
	TargetID   string
	StartTime  *time.Time
	EndTime    *time.Time
	Pagination PaginationParams
}

type AuditLogEntry struct {
	LogID         string
	Action        AuditAction
	Operator      UserInfo
	TargetType    AuditTargetType
	TargetID      string
	TargetName    string
	Details       AuditDetails
	IPLocation    string
	IPAddress     string
	DeviceSummary string
	CreatedAt     time.Time
}

type auditLogPage struct {
	Items []AuditLogItem
	Next  *auditCursor
}

type adminUserCursor struct {
	CreatedAt time.Time `json:"created_at"`
	UserID    string    `json:"user_id"`
}

type adminUserPage struct {
	Items []AdminUserItem
	Next  *adminUserCursor
}

type adminGroupCursor struct {
	CreatedAt time.Time `json:"created_at"`
	GroupID   string    `json:"group_id"`
}

type adminGroupPage struct {
	Items []AdminGroupItem
	Next  *adminGroupCursor
}

type adminReportCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ReportID  string    `json:"report_id"`
}

type adminReportPage struct {
	Items []AdminReportItem
	Next  *adminReportCursor
}

type UserBanState struct {
	Status       UserAccountStatus
	Reason       string
	DurationDays int
	BannedAt     time.Time
}

type reportHandleRecord struct {
	Item       AdminReportItem
	TargetUser usersmodule.UserInfo
}

func reportItemFromModule(item reportsmodule.ReportItem, reporter usersmodule.UserInfo, handledBy *usersmodule.UserInfo, handleNote string) AdminReportItem {
	return AdminReportItem{
		ReportID:    item.ReportID,
		TargetType:  ReportTargetType(item.TargetType),
		TargetID:    item.TargetID,
		Reason:      string(item.Reason),
		Description: item.Description,
		Status:      ReportStatus(item.Status),
		CreatedAt:   item.CreatedAt,
		Reporter:    reporter,
		HandledBy:   handledBy,
		HandleNote:  handleNote,
	}
}
