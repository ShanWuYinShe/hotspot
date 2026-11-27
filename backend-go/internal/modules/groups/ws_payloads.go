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

// GroupAnnouncementCreatedPayload 群公告发布事件（spec api_spec_websocket.md:
// `{ announcement: GroupAnnouncementItem }`）：此前实现把 GroupAnnouncementItem
// 的字段拍平到 payload 顶层且缺 content/is_pinned，与 spec 形状不一致（契约审计 B05）。
type GroupAnnouncementCreatedPayload struct {
	Announcement GroupAnnouncementItem `json:"announcement"`
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
	// MemberCount 退群后的剩余成员总数（spec 必填）：客户端据此刷新群徽标，
	// 缺键会读到 undefined（契约审计 B03）。
	MemberCount int `json:"member_count"`
}

// GroupMemberRoleChangedPayload 角色变更事件（spec → GroupMemberRoleChangedPayload）：
// 恒在 old_role/new_role 两个字段，客户端据此判断升/降级并渲染提示文案。
// 改动前只发单值 role，无法区分升级还是降级（契约审计 B02）。
type GroupMemberRoleChangedPayload struct {
	GroupID   string         `json:"group_id"`
	User      types.UserInfo `json:"user"`
	OldRole   string         `json:"old_role"`
	NewRole   string         `json:"new_role"`
	ChangedBy types.UserInfo `json:"changed_by"`
	ChangedAt string         `json:"changed_at"`
}

// GroupMemberMutedPayload 禁言事件（spec → GroupMemberMutedPayload）：
// muted_until 必须与 HTTP GroupViewerState.muted_until 同口径（同一截止时间），
// 客户端无需再用 duration_minutes 自行推算（契约审计 B04）。
type GroupMemberMutedPayload struct {
	GroupID         string         `json:"group_id"`
	User            types.UserInfo `json:"user"`
	MutedBy         types.UserInfo `json:"muted_by"`
	MutedUntil      string         `json:"muted_until"`
	DurationMinutes int            `json:"duration_minutes"`
	MutedAt         string         `json:"muted_at"`
}

type GroupMemberUnmutedPayload struct {
	GroupID   string         `json:"group_id"`
	User      types.UserInfo `json:"user"`
	UnmutedBy types.UserInfo `json:"unmuted_by"`
	UnmutedAt string         `json:"unmuted_at"`
}

// GroupMemberRemovedPayload 成员被移除事件（spec → GroupMemberRemovedPayload）：
// user/removed_by 均为必填 UserInfo（仅 reason 可空）。此前本人动作路径不写键，
// 客户端读 user/removed_by 得到 undefined（契约审计 B15）；管理员操作时
// removed_by 即操作者，本人动作路径由调用方显式填同一 UserInfo。
type GroupMemberRemovedPayload struct {
	GroupID     string         `json:"group_id"`
	User        types.UserInfo `json:"user"`
	RemovedBy   types.UserInfo `json:"removed_by"`
	Reason      string         `json:"reason"`
	RemovedAt   string         `json:"removed_at"`
	MemberCount int            `json:"member_count"`
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

// GroupApplicationReceivedPayload：message 为 spec 恒定 nullable（string | null），
// 无附言时必须显式下发 null（去掉 omitempty，赋值侧已保证空串置 nil）。
type GroupApplicationReceivedPayload struct {
	NotifyID      string         `json:"notify_id"`
	ApplicationID *string        `json:"application_id"`
	GroupID       string         `json:"group_id"`
	Applicant     types.UserInfo `json:"applicant"`
	CreatedAt     string         `json:"created_at"`
	Message       *string        `json:"message"`
}

// GroupApplicationCancelledPayload 申请被申请人取消事件（spec →
// GroupApplicationCancelledPayload）：reason 为恒定 nullable（null = 无理由），
// 无理由时必须显式下发 null（契约审计 B08）。
type GroupApplicationCancelledPayload struct {
	ApplicationID string         `json:"application_id"`
	GroupID       string         `json:"group_id"`
	Applicant     types.UserInfo `json:"applicant"`
	CancelledAt   string         `json:"cancelled_at"`
	Reason        *string        `json:"reason"`
}

// GroupInviteReceivedPayload：邀请附言 message 为 spec 恒定 nullable（null = 无附言），
// 此前用 map 条件写键，无附言时整键缺失（B08 同族补齐）。
type GroupInviteReceivedPayload struct {
	NotifyID  string            `json:"notify_id"`
	InviteID  string            `json:"invite_id"`
	Group     GroupBriefPayload `json:"group"`
	Actor     types.UserInfo    `json:"actor"`
	Message   *string           `json:"message"`
	CreatedAt string            `json:"created_at"`
}

// GroupApplicationRejectedPayload：拒绝理由 reason 为 spec 恒定 nullable。
type GroupApplicationRejectedPayload struct {
	NotifyID      string            `json:"notify_id"`
	ApplicationID string            `json:"application_id"`
	Group         GroupBriefPayload `json:"group"`
	RejectedAt    string            `json:"rejected_at"`
	Reason        *string           `json:"reason"`
}

// GroupBriefPayload：GroupApplicationResponse 内嵌的群摘要。
type GroupBriefPayload struct {
	GroupID      string  `json:"group_id"`
	Name         string  `json:"name"`
	AvatarURL    *string `json:"avatar_url"`
	MembersCount int     `json:"members_count"`
}

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
	Reason    *string        `json:"reason"` // spec 恒定 nullable（null = 无理由）
}
