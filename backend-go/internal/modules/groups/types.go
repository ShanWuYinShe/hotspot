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

// GroupViewerState is the viewer's group member state (GroupItem.viewer_state).
// A nil viewer_state means the viewer is not a member. MutedUntil is a truthy
// nullable (null = not muted) and is only visible to the muted member
// themselves. spec api_spec_types.md GroupViewerState.
type GroupViewerState struct {
	Role        GroupRole `json:"role"`
	UnreadCount int       `json:"unread_count"`
	IsMuted     bool      `json:"is_muted"`
	MutedUntil  *string   `json:"muted_until"`
	IsPinned    bool      `json:"is_pinned"`
	JoinedAt    string    `json:"joined_at"`
}

// GroupSettings 是群组管理设置（仅群管理守卫子树 groups/admin/get_settings
// 响应；create/update 请求同构）。spec api_spec_types.md GroupSettings。
// 群管理字段不再随 GroupItem 下发，统一经本类型获取。
type GroupSettings struct {
	JoinMethod        GroupJoinMethod `json:"join_method"`
	RequiresApproval  bool            `json:"requires_approval"`
	HasPassword       bool            `json:"has_password"`
	IsSearchable      bool            `json:"is_searchable"`
	MembersSearchable bool            `json:"members_searchable"`
	MaxMembers        int             `json:"max_members"`
	Location          Location        `json:"location"`
}

// GroupItem 是群组统一公开类型（搜索/详情/Feed；对所有人形状恒定）。
// 群管理字段（has_password/is_searchable/members_searchable/max_members）不在本类型，
// 见 GroupSettings（groups/admin/get_settings）。spec api_spec_types.md GroupItem。
//
// HasPassword/IsSearchable/MembersSearchable/MaxMembers 仍是服务内部用于准入/
// 搜索/设置的存储字段，但不对外序列化（json:"-"），避免泄露群管理信息。
type GroupItem struct {
	Group            GroupInfo         `json:"group"`
	Description      string            `json:"description,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	JoinMethod       GroupJoinMethod   `json:"join_method,omitempty"`
	RequiresApproval bool              `json:"requires_approval"`
	// 群管理字段（内部存储/准入逻辑使用，不对外序列化）
	HasPassword       bool   `json:"-"`
	IsSearchable      bool   `json:"-"`
	MembersSearchable bool   `json:"-"`
	MaxMembers        int    `json:"-"`
	Location          RelativeLocation  `json:"location"`
	ViewerState       *GroupViewerState `json:"viewer_state"`
	FriendsInGroup    *int              `json:"friends_in_group"`
	OnlineMembersCount *int              `json:"online_members_count"`
	NextActivity      *string           `json:"next_activity"`
	LastActivityAt    *string           `json:"last_activity_at"`
	CreatedAt         *string           `json:"created_at"`
	OwnerID           *string           `json:"owner_id"`
	LastMessage       *GroupLastMessage `json:"last_message"`
}

// GroupLastMessage 群列表最后一条消息预览（仅 groups/list_joined 返回，见 api_spec_types.md GroupJoinedItem）。
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
	JoinMethod        GroupJoinMethod `json:"join_method,omitempty"`
	RequiresApproval  *bool           `json:"requires_approval,omitempty"`
	IsSearchable      *bool           `json:"is_searchable,omitempty"`
	MembersSearchable *bool           `json:"members_searchable,omitempty"` // 默认 true
	MaxMembers        int             `json:"max_members,omitempty"`
	Location          Location        `json:"location"`
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
	JoinMethod        *GroupJoinMethod `json:"join_method,omitempty"`
	RequiresApproval  *bool            `json:"requires_approval,omitempty"`
	NextActivity      *string          `json:"next_activity,omitempty"`
	IsSearchable      *bool            `json:"is_searchable,omitempty"`
	MembersSearchable *bool            `json:"members_searchable,omitempty"`
	MaxMembers        *int             `json:"max_members,omitempty"`
	Location          *Location        `json:"location,omitempty"`
}

// GetGroupSettingsRequest fetches the admin-visible group settings
// (groups/admin/get_settings, guard subtree).
type GetGroupSettingsRequest struct {
	GroupID string `json:"group_id"`
}

// DismissGroupRequest dismisses a group (owner only). Reason is audit-only:
// it is never surfaced to members (they receive the owner_dismissed category,
// spec api_spec_group.md).
type DismissGroupRequest struct {
	GroupID string  `json:"group_id"`
	Reason  *string `json:"reason,omitempty"`
}

type GroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// ListAdminGroupMembersRequest lists members from the admin perspective
// (groups/admin/members/list, returns AdminGroupMemberItem with truthy
// muted_until). spec api_spec_group.md.
type ListAdminGroupMembersRequest struct {
	GroupID    string            `json:"group_id"`
	Role       *GroupRole        `json:"role,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// GroupMemberItem 是公开成员列表项（members/list 与 members/search 共用）。
// 公开视角不携带禁言信息（muted_until 归位到 AdminGroupMemberItem 与
// GroupViewerState）。presence 字段为隐私受控恒定 nullable。spec api_spec_types.md。
type GroupMemberItem struct {
	User           UserInfo            `json:"user"`
	Role           GroupRole           `json:"role"`
	JoinedAt       string              `json:"joined_at"`
	PresenceStatus *UserPresenceStatus `json:"presence_status"`
	LastOnline     *string             `json:"last_online_at"`
}

// AdminGroupMemberItem 是管理视角成员列表项（仅 groups/admin/members/list
// 返回；muted_until 为真值，null=未禁言）。spec api_spec_types.md。
type AdminGroupMemberItem struct {
	User       UserInfo  `json:"user"`
	Role       GroupRole `json:"role"`
	JoinedAt   string    `json:"joined_at"`
	MutedUntil *string   `json:"muted_until"`
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

// CreateGroupInviteRequest 是守卫子树 groups/admin/invites/create 的请求类型。
type CreateGroupInviteRequest struct {
	GroupID string  `json:"group_id"`
	UserID  string  `json:"user_id"`
	Message *string `json:"message,omitempty"`
}

// InviteMemberRequest 是 repo/service 内部沿用类型（与 CreateGroupInviteRequest 同构）。
type InviteMemberRequest = CreateGroupInviteRequest

// Invitation：邀请响应（spec group.md → members/invite）
// invite_id/expires_at 可选：被邀请者已是成员（no-op 成功）时省略；TTL=0（永不过期）时 expires_at 省略
type Invitation struct {
	InviteID  string `json:"invite_id,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ListSentInvitesRequest 是守卫子树 groups/admin/invites/list_sent 的请求类型。
type ListSentInvitesRequest struct {
	GroupID    string             `json:"group_id"`
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

// GroupSentInvitesRequest 是内部沿用类型。
type GroupSentInvitesRequest = ListSentInvitesRequest

type GroupSentInviteItem struct {
	InviteID    string            `json:"invite_id"`
	Invitee     UserInfo          `json:"invitee"`
	Status      ApplicationStatus `json:"status"`
	Message     *string           `json:"message,omitempty"`
	CreatedAt   string            `json:"created_at"`
	ExpiresAt   *string           `json:"expires_at,omitempty"`
	RespondedAt *string           `json:"responded_at,omitempty"`
}

// RevokeGroupInviteRequest 是守卫子树 groups/admin/invites/revoke 的请求类型。
type RevokeGroupInviteRequest struct {
	GroupID  string  `json:"group_id"`
	InviteID string  `json:"invite_id"`
	Reason   *string `json:"reason,omitempty"`
}

// RevokeSentInviteRequest 是内部沿用类型。
type RevokeSentInviteRequest = RevokeGroupInviteRequest

type SetMemberRoleRequest struct {
	GroupID string    `json:"group_id"`
	UserID  string    `json:"user_id"`
	Role    GroupRole `json:"role"`
}

// SetMuteRequest 是两态统一禁言端点 groups/admin/members/set_mute 的请求类型
// （契约级）。active=true 时 duration_minutes 必填（1-43200）；active=false 时忽略。
type SetMuteRequest struct {
	GroupID         string `json:"group_id"`
	UserID          string `json:"user_id"`
	Active          bool   `json:"active"`
	DurationMinutes *int   `json:"duration_minutes,omitempty"`
}

// MuteMemberRequest / UnmuteMemberRequest 为 repo/service 内部请求结构
// （service 按 SetMuteRequest.Active 拆分后传入 repo），不对外暴露。
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

// SetMemberBanRequest 是两态统一端点 groups/admin/members/set_ban 的请求类型（契约级）。
type SetMemberBanRequest struct {
	GroupID string  `json:"group_id"`
	UserID  string  `json:"user_id"`
	Active  bool    `json:"active"`
	Reason  *string `json:"reason,omitempty"`
}

// BanMemberRequest / UnbanMemberRequest 为 repo 层内部请求结构（service.SetMemberBan
// 按 Active 拆分后传入 repo），不对外暴露。
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

// GroupApplicationItem 是群组申请列表项（群管理员视角与申请人视角共用此类型）。
// F7 对称化后不再暴露审批者身份（HandledBy 已删除）。spec api_spec_types.md。
type GroupApplicationItem struct {
	ApplicationID string            `json:"application_id"`
	GroupID       string            `json:"-"`
	Applicant     UserInfo          `json:"applicant"`
	Message       *string           `json:"message,omitempty"`
	Status        ApplicationStatus `json:"status"`
	CreatedAt     string            `json:"created_at"`
	HandledAt     *string           `json:"handled_at,omitempty"`
}

// ApproveGroupApplicationRequest approves a group application. GroupID is the
// group-level auth anchor and must match the application's group, otherwise
// the request fails (spec api_spec_group.md).
type ApproveGroupApplicationRequest struct {
	GroupID       string `json:"group_id"`
	ApplicationID string `json:"application_id"`
}

type RejectGroupApplicationRequest struct {
	GroupID       string  `json:"group_id"`
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

type UpdateGroupPreferencesRequest struct {
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

// RecallGroupMessageAdminRequest 是守卫子树 groups/admin/messages/recall 的请求类型。
// 管理员撤回任意成员消息，无时间窗限制。spec api_spec_group.md。
type RecallGroupMessageAdminRequest struct {
	GroupID   string `json:"group_id"`
	MessageID string `json:"message_id"`
}

// ListJoinedGroupsRequest（groups/list_joined，成员视角）。spec api_spec_group.md。
type ListJoinedGroupsRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// MyGroupsRequest 是内部沿用类型。
type MyGroupsRequest = ListJoinedGroupsRequest

// ListSubmittedApplicationsRequest（groups/applications/list_submitted，申请人视角）。
type ListSubmittedApplicationsRequest struct {
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

// MyApplicationsRequest 是内部沿用类型。
type MyApplicationsRequest = ListSubmittedApplicationsRequest

// CancelApplicationRequest cancels a pending group application. GroupID must
// match the application's group; Reason (0-200 chars) is delivered to the
// group admins, aligned with the admin-side reject reason.
type CancelApplicationRequest struct {
	GroupID       string  `json:"group_id"`
	ApplicationID string  `json:"application_id"`
	Reason        *string `json:"reason,omitempty"`
}

// ListReceivedInvitesRequest（groups/invites/list_received，受邀人视角）。
type ListReceivedInvitesRequest struct {
	Status     *ApplicationStatus `json:"status,omitempty"`
	Pagination *PaginationParams  `json:"pagination,omitempty"`
}

// MyInvitesRequest 是内部沿用类型。
type MyInvitesRequest = ListReceivedInvitesRequest

type GroupInviteItem struct {
	InviteID    string            `json:"invite_id"`
	Group       GroupInfo         `json:"group"`
	Inviter     UserInfo          `json:"inviter"`
	Message     *string           `json:"message,omitempty"`
	Status      ApplicationStatus `json:"status"`
	CreatedAt   string            `json:"created_at"`
	ExpiresAt   *string           `json:"expires_at,omitempty"`
	RespondedAt *string           `json:"responded_at,omitempty"`
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

// PaginatedGroups 通用分页容器（响应侧恒定 nullable：next_cursor 恒在，null=没有更多）。
type PaginatedGroups struct {
	Items      []GroupItem `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}

type PaginatedGroupMembers struct {
	Items      []GroupMemberItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type PaginatedAdminGroupMembers struct {
	Items      []AdminGroupMemberItem `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}

type PaginatedSentInvites struct {
	Items      []GroupSentInviteItem `json:"items"`
	NextCursor *string               `json:"next_cursor"`
}

type PaginatedBannedMembers struct {
	Items      []BannedMemberItem `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

type PaginatedApplications struct {
	Items      []GroupApplicationItem `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}

type PaginatedInvites struct {
	Items      []GroupInviteItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
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
