package groups

import (
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

type GroupJoinMethod string

type GroupRole string

type ApplicationStatus string

type JoinGroupStatus string

// PaginationParams aliases the shared request-pagination shape
// (api_spec_base.md PaginationParams) so group request bodies use the nested
// {"pagination":{"limit","cursor"}} shape the spec and clients expect
// (previously these were flat top-level limit/cursor fields).
type PaginationParams = types.PaginationParams

// paginationLimit returns the page size, defaulting to fallback when unset or
// out of range. Free function because methods cannot be defined on a type
// aliased from another package.
func paginationLimit(p *PaginationParams, fallback, max int) int {
	if p == nil || p.Limit <= 0 {
		return fallback
	}
	if p.Limit > max {
		return max
	}
	return p.Limit
}

// paginationCursor returns the pagination cursor (empty when unset).
func paginationCursor(p *PaginationParams) string {
	if p == nil {
		return ""
	}
	return p.Cursor
}

const (
	GroupJoinMethodDirect     GroupJoinMethod = "direct"
	GroupJoinMethodPassword   GroupJoinMethod = "password"
	GroupJoinMethodInviteOnly GroupJoinMethod = "invite_only"

	GroupRoleOwner  GroupRole = "owner"
	GroupRoleAdmin  GroupRole = "admin"
	GroupRoleMember GroupRole = "member"

	ApplicationStatusPending  ApplicationStatus = "pending"
	ApplicationStatusAccepted ApplicationStatus = "accepted"
	ApplicationStatusRejected ApplicationStatus = "rejected"
	ApplicationStatusRevoked  ApplicationStatus = "revoked"

	JoinGroupStatusJoined          JoinGroupStatus = "joined"
	JoinGroupStatusPendingApproval JoinGroupStatus = "pending_approval"
)

type GroupInfo struct {
	GroupID      string `json:"group_id"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	MembersCount int    `json:"members_count"`
}

type Location struct {
	Point struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"point"`
	LocationName   string   `json:"location_name"`
	Distance       *float64 `json:"distance,omitempty"`
	BearingDegrees *float64 `json:"bearing_degrees,omitempty"`
}

type RelativeLocation = Location

type MyGroupState struct {
	Role        GroupRole `json:"role"`
	IsJoined    bool      `json:"is_joined"`
	UnreadCount int       `json:"unread_count"`
	IsMuted     bool      `json:"is_muted"`
	MutedUntil  string    `json:"muted_until,omitempty"`
	IsPinned    bool      `json:"is_pinned"`
	JoinedAt    string    `json:"joined_at,omitempty"`
}

type GroupItem struct {
	Group              GroupInfo         `json:"group"`
	Description        string            `json:"description,omitempty"`
	Tags               []string          `json:"tags,omitempty"`
	JoinMethod         GroupJoinMethod   `json:"join_method,omitempty"`
	RequiresApproval   bool              `json:"requires_approval"`
	HasPassword        bool              `json:"has_password,omitempty"`
	Location           RelativeLocation  `json:"location"`
	MyState            *MyGroupState     `json:"my_state,omitempty"`
	FriendsInGroup     int               `json:"friends_in_group,omitempty"`
	OnlineMembersCount *int              `json:"online_members_count,omitempty"`
	NextActivity       *string           `json:"next_activity,omitempty"`
	IsSearchable       bool              `json:"is_searchable,omitempty"`
	MaxMembers         int               `json:"max_members,omitempty"`
	LastActivityAt     *string           `json:"last_activity_at,omitempty"`
	CreatedAt          *string           `json:"created_at,omitempty"`
	OwnerID            string            `json:"owner_id,omitempty"`
	LastMessage        *GroupLastMessage `json:"last_message,omitempty"`
}

// GroupLastMessage 群列表最后一条消息预览（仅 groups/list_my 返回，见 api_spec_types.md GroupItem）。
type GroupLastMessage struct {
	MessageID string   `json:"message_id"`
	Type      string   `json:"type"`
	Content   string   `json:"content,omitempty"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type CreateGroupRequest struct {
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	AvatarFileID     string          `json:"avatar_file_id,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	Password         string          `json:"password,omitempty"`
	JoinMethod       GroupJoinMethod `json:"join_method,omitempty"`
	RequiresApproval *bool           `json:"requires_approval,omitempty"`
	IsSearchable     *bool           `json:"is_searchable,omitempty"`
	MaxMembers       int             `json:"max_members,omitempty"`
	Location         Location        `json:"location"`
}

type GroupOpenTheDoorRequest struct {
	GroupID string `json:"group_id"`
}

type SearchGroupsRequest struct {
	Keyword        string            `json:"keyword,omitempty"`
	Tag            string            `json:"tag,omitempty"`
	Radius         *int              `json:"radius,omitempty"`
	SortBy         string            `json:"sort_by,omitempty"`
	ReferencePoint GeoPoint          `json:"reference_point"`
	Pagination     *PaginationParams `json:"pagination,omitempty"`
}

type GeoPoint = types.GeoPoint

type UpdateGroupRequest struct {
	GroupID          string           `json:"group_id"`
	Name             string           `json:"name,omitempty"`
	Description      string           `json:"description,omitempty"`
	AvatarFileID     string           `json:"avatar_file_id,omitempty"`
	Tags             []string         `json:"tags,omitempty"`
	Password         *string          `json:"password,omitempty"`
	ClearPassword    *bool            `json:"clear_password,omitempty"`
	JoinMethod       *GroupJoinMethod `json:"join_method,omitempty"`
	RequiresApproval *bool            `json:"requires_approval,omitempty"`
	NextActivity     *string          `json:"next_activity,omitempty"`
	IsSearchable     *bool            `json:"is_searchable,omitempty"`
	MaxMembers       *int             `json:"max_members,omitempty"`
	Location         *Location        `json:"location,omitempty"`
}

type DismissGroupRequest struct {
	GroupID string `json:"group_id"`
}

type GroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type GroupMemberItem struct {
	User           UserInfo            `json:"user"`
	Role           GroupRole           `json:"role"`
	JoinedAt       string              `json:"joined_at"`
	MutedUntil     *string             `json:"muted_until,omitempty"`
	PresenceStatus *UserPresenceStatus `json:"presence_status,omitempty"`
	LastOnline     *string             `json:"last_online,omitempty"`
}

type UserPresenceStatus string

const (
	UserPresenceOnline  UserPresenceStatus = "online"
	UserPresenceOffline UserPresenceStatus = "offline"
)

type UserInfo = types.UserInfo

type SearchGroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Keyword    string            `json:"keyword"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type InviteMemberRequest struct {
	GroupID string  `json:"group_id"`
	UserID  string  `json:"user_id"`
	Message *string `json:"message,omitempty"`
}

type Invitation struct {
	InviteID  string `json:"invite_id"`
	ExpiresAt string `json:"expires_at"`
}

type GroupSentInvitesRequest struct {
	GroupID    string             `json:"group_id"`
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

type GroupSentInviteItem struct {
	InviteID    string            `json:"invite_id"`
	Invitee     UserInfo          `json:"invitee"`
	Status      ApplicationStatus `json:"status"`
	Message     *string           `json:"message,omitempty"`
	CreatedAt   string            `json:"created_at"`
	ExpiresAt   *string           `json:"expires_at,omitempty"`
	RespondedAt *string           `json:"responded_at,omitempty"`
}

type RevokeSentInviteRequest struct {
	GroupID  string  `json:"group_id"`
	InviteID string  `json:"invite_id"`
	Reason   *string `json:"reason,omitempty"`
}

type SetMemberRoleRequest struct {
	GroupID string    `json:"group_id"`
	UserID  string    `json:"user_id"`
	Role    GroupRole `json:"role"`
}

type MuteMemberRequest struct {
	GroupID         string `json:"group_id"`
	UserID          string `json:"user_id"`
	Active          bool   `json:"active"`
	DurationMinutes int    `json:"duration_minutes"`
}

type UnmuteMemberRequest struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
}

type BanMemberRequest struct {
	GroupID string  `json:"group_id"`
	UserID  string  `json:"user_id"`
	Reason  *string `json:"reason,omitempty"`
}

type UnbanMemberRequest struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
}

type BannedMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type BannedMemberItem struct {
	User     UserInfo  `json:"user"`
	BannedAt string    `json:"banned_at"`
	Reason   *string   `json:"reason,omitempty"`
	BannedBy *UserInfo `json:"banned_by,omitempty"`
}

type RemoveMemberRequest struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
}

type GroupApplicationsRequest struct {
	GroupID    string             `json:"group_id"`
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

type GroupApplicationItem struct {
	ApplicationID string            `json:"application_id"`
	GroupID       string            `json:"-"`
	Applicant     UserInfo          `json:"applicant"`
	Message       *string           `json:"message,omitempty"`
	Status        ApplicationStatus `json:"status"`
	CreatedAt     string            `json:"created_at"`
	HandledAt     *string           `json:"handled_at,omitempty"`
	HandledBy     *UserInfo         `json:"handled_by,omitempty"`
}

type ApproveGroupApplicationRequest struct {
	ApplicationID string `json:"application_id"`
}

type RejectGroupApplicationRequest struct {
	ApplicationID string  `json:"application_id"`
	Reason        *string `json:"reason,omitempty"`
}

type CreateGroupAnnouncementRequest struct {
	GroupID  string `json:"group_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	IsPinned *bool  `json:"is_pinned,omitempty"`
}

type GroupAnnouncementItem struct {
	AnnouncementID string   `json:"announcement_id"`
	GroupID        string   `json:"group_id"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	CreatedBy      UserInfo `json:"created_by"`
	CreatedAt      string   `json:"created_at"`
	IsPinned       bool     `json:"is_pinned"`
}

type GroupAnnouncementsRequest struct {
	GroupID string `json:"group_id"`
}

type UpdateGroupAnnouncementRequest struct {
	GroupID        string  `json:"group_id"`
	AnnouncementID string  `json:"announcement_id"`
	Title          *string `json:"title,omitempty"`
	Content        *string `json:"content,omitempty"`
	IsPinned       *bool   `json:"is_pinned,omitempty"`
}

type DeleteGroupAnnouncementRequest struct {
	GroupID        string `json:"group_id"`
	AnnouncementID string `json:"announcement_id"`
}

type UpdateGroupMySettingsRequest struct {
	GroupID  string `json:"group_id"`
	IsMuted  *bool  `json:"is_muted,omitempty"`
	IsPinned *bool  `json:"is_pinned,omitempty"`
}

type JoinGroupRequest struct {
	GroupID  string  `json:"group_id"`
	Password *string `json:"password,omitempty"`
	Message  *string `json:"message,omitempty"`
}

type JoinGroupResult struct {
	Status      JoinGroupStatus `json:"status"`
	Application *struct {
		ApplicationID string `json:"application_id"`
	} `json:"application,omitempty"`
	// IsNewApplication 为内部标记（不序列化）：approval 路径是否本次新创建的申请。
	// 命中已有 pending 申请（自然去重，见 api_spec_base.md）时为 false，
	// service 据此跳过重复通知管理员。
	IsNewApplication bool `json:"-"`
}

type TransferOwnershipRequest struct {
	GroupID    string `json:"group_id"`
	NewOwnerID string `json:"new_owner_id"`
}

type MyGroupsRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type MyApplicationsRequest struct {
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

type CancelApplicationRequest struct {
	ApplicationID string `json:"application_id"`
}

type MyInvitesRequest struct {
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

type GroupInviteItem struct {
	InviteID  string            `json:"invite_id"`
	Group     GroupInfo         `json:"group"`
	Inviter   UserInfo          `json:"inviter"`
	Message   *string           `json:"message,omitempty"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt string            `json:"created_at"`
	ExpiresAt *string           `json:"expires_at,omitempty"`
}

type AcceptInviteRequest struct {
	InviteID string `json:"invite_id"`
}

type RejectInviteRequest struct {
	InviteID string  `json:"invite_id"`
	Reason   *string `json:"reason,omitempty"`
}

type LeaveGroupRequest struct {
	GroupID string `json:"group_id"`
}

type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type PaginatedGroupMembers struct {
	Items      []GroupMemberItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type PaginatedSentInvites struct {
	Items      []GroupSentInviteItem `json:"items"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

type PaginatedBannedMembers struct {
	Items      []BannedMemberItem `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type PaginatedApplications struct {
	Items      []GroupApplicationItem `json:"items"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type PaginatedInvites struct {
	Items      []GroupInviteItem `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// myGroupsCursor is the keyset cursor for ListMyGroups, ordered by
// (joined_at DESC, group_id DESC). It is opaque to clients — encoded with the
// shared AES-GCM cursor codec under the "groups.my_groups" namespace.
type myGroupsCursor struct {
	JoinedAt string `json:"joined_at"`
	GroupID  string `json:"group_id"`
}

const myGroupsCursorNamespace = "groups.my_groups"

func encodeMyGroupsCursor(c myGroupsCursor) string {
	value, err := cursor.Encode(myGroupsCursorNamespace, c)
	if err != nil {
		return ""
	}
	return value
}
