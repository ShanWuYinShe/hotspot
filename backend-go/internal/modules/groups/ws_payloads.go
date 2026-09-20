package groups

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — groups 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 各事件负载的实现侧镜像）。此前为 map[string]any
// 字面量，字段名拼写错误无法被编译器发现；struct 化后字段漂移在编译期暴露。
//
// omitempty 与原 map 的「条件写入键」行为一一对应（条件注入的键带
// omitempty，恒在键不带），序列化结果与改造前完全一致。

type GroupAnnouncementCreatedPayload struct {
	GroupID        string         `json:"group_id"`
	AnnouncementID string         `json:"announcement_id"`
	Title          string         `json:"title"`
	CreatedBy      types.UserInfo `json:"created_by"`
	CreatedAt      string         `json:"created_at"`
}

type GroupMemberJoinedPayload struct {
	GroupID     string         `json:"group_id"`
	User        types.UserInfo `json:"user"`
	Role        string         `json:"role"`
	MemberCount int            `json:"member_count"`
	JoinedAt    string         `json:"joined_at"`
}

type GroupMemberLeftPayload struct {
	GroupID string         `json:"group_id"`
	User    types.UserInfo `json:"user"`
	LeftAt  string         `json:"left_at"`
}

type GroupMemberRoleChangedPayload struct {
	GroupID   string         `json:"group_id"`
	User      types.UserInfo `json:"user"`
	Role      string         `json:"role"`
	ChangedBy types.UserInfo `json:"changed_by"`
	ChangedAt string         `json:"changed_at"`
}

type GroupMemberMutedPayload struct {
	GroupID         string         `json:"group_id"`
	User            types.UserInfo `json:"user"`
	MutedBy         types.UserInfo `json:"muted_by"`
	DurationMinutes int            `json:"duration_minutes"`
	MutedAt         string         `json:"muted_at"`
}

type GroupMemberUnmutedPayload struct {
	GroupID   string         `json:"group_id"`
	User      types.UserInfo `json:"user"`
	UnmutedBy types.UserInfo `json:"unmuted_by"`
	UnmutedAt string         `json:"unmuted_at"`
}

// GroupMemberRemovedPayload：user/removed_by 在「本人退群/禁用触发」路径
// 缺省（原 map 未写键），故为指针 + omitempty。
type GroupMemberRemovedPayload struct {
	GroupID     string          `json:"group_id"`
	User        *types.UserInfo `json:"user,omitempty"`
	RemovedBy   *types.UserInfo `json:"removed_by,omitempty"`
	Reason      string          `json:"reason"`
	RemovedAt   string          `json:"removed_at"`
	MemberCount int             `json:"member_count"`
}

type GroupDismissedPayload struct {
	GroupID     string `json:"group_id"`
	DismissedAt string `json:"dismissed_at"`
	Reason      string `json:"reason"`
}

type GroupInviteExpiredPayload struct {
	InviteID  string `json:"invite_id"`
	GroupID   string `json:"group_id"`
	ExpiredAt string `json:"expired_at"`
}

type GroupInviteResponsePayload struct {
	InviteID    string         `json:"invite_id"`
	GroupID     string         `json:"group_id"`
	User        types.UserInfo `json:"user"`
	RespondedAt string         `json:"responded_at"`
}

type GroupApplicationExpiredPayload struct {
	ApplicationID string         `json:"application_id"`
	GroupID       string         `json:"group_id"`
	Applicant     types.UserInfo `json:"applicant"`
	ExpiredAt     string         `json:"expired_at"`
}

// GroupApplicationReceivedPayload：application_id 为 *string（可 nil，键恒在
// 对齐原 map）；message 原逻辑「非空才写键」，字段赋值侧保证空串置 nil。
type GroupApplicationReceivedPayload struct {
	NotifyID      string         `json:"notify_id"`
	ApplicationID *string        `json:"application_id"`
	GroupID       string         `json:"group_id"`
	Applicant     types.UserInfo `json:"applicant"`
	CreatedAt     string         `json:"created_at"`
	Message       *string        `json:"message,omitempty"`
}

// GroupBriefPayload：GroupApplicationResponse 内嵌的群摘要。
type GroupBriefPayload struct {
	GroupID      string  `json:"group_id"`
	Name         string  `json:"name"`
	AvatarURL    *string `json:"avatar_url"`
	MembersCount int     `json:"members_count"`
}

// userInfoPtr 把值语义的 UserInfo 转为指针（GroupMemberRemovedPayload 的
// user/removed_by 在本人动作路径缺省，需要指针 + omitempty 才能省略键）。
func userInfoPtr(u types.UserInfo) *types.UserInfo { return &u }

type GroupApplicationResponsePayload struct {
	NotifyID      string            `json:"notify_id"`
	ApplicationID string            `json:"application_id"`
	Group         GroupBriefPayload `json:"group"`
	RespondedAt   string            `json:"responded_at"`
}

type GroupUpdatedPayload struct {
	GroupID   string         `json:"group_id"`
	UpdatedBy types.UserInfo `json:"updated_by"`
	UpdatedAt string         `json:"updated_at"`
	// UpdatedFields 变更字段 diff。键属于固定字段集，但「仅变化字段」的动态
	// 语义无法用单个 struct + omitempty 等价表达（bool false / 空 slice 的
	// 合法变更会被 omitempty 吞掉），故保留 map——本模块唯一登记例外，
	// 生成逻辑见 buildGroupUpdatedFields。
	UpdatedFields map[string]any `json:"updated_fields,omitempty"`
}

type GroupInviteRevokedPayload struct {
	InviteID  string         `json:"invite_id"`
	GroupID   string         `json:"group_id"`
	RevokedBy types.UserInfo `json:"revoked_by"`
	RevokedAt string         `json:"revoked_at"`
	Reason    *string        `json:"reason,omitempty"` // 原逻辑：非空才写键
}
