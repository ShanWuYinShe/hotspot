package groupmessages

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — groupmessages 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。struct 化后字段漂移在编译期暴露。

// GroupMessageSentPayload 是群聊新消息事件负载（spec →
// GroupMessageNewPayload）。content/share_code/reply_to/media 为 spec「恒定
// nullable」（无值显式 null），mentions 恒为数组（空数组 = 无 @）——此前用
// map 条件写键，纯文本帧缺 media/reply_to、无 @ 帧缺 mentions（契约审计 B07）。
type GroupMessageSentPayload struct {
	MessageID string         `json:"message_id"`
	GroupID   string         `json:"group_id"`
	Sender    types.UserInfo `json:"sender"`
	Type      string         `json:"type"`
	Content   *string        `json:"content"`
	ShareCode *string        `json:"share_code"`
	ReplyTo   *MessageBrief  `json:"reply_to"`
	Mentions  []string       `json:"mentions"`
	Media     any            `json:"media"`
	SentAt    string         `json:"sent_at"`
}

type GroupMessageMentionedPayload struct {
	NotifyID       string         `json:"notify_id"`
	GroupID        string         `json:"group_id"`
	GroupName      string         `json:"group_name"`
	MessageID      string         `json:"message_id"`
	Actor          types.UserInfo `json:"actor"`
	ContentPreview string         `json:"content_preview"`
	MentionedAt    string         `json:"mentioned_at"`
}

type GroupMessageEditedPayload struct {
	GroupID   string `json:"group_id"`
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	EditedAt  string `json:"edited_at"`
}

// RecalledByUser 群消息撤回者（spec 仅含 user_id 的嵌套对象）。
type RecalledByUser struct {
	UserID string `json:"user_id"`
}

type GroupMessageRecalledPayload struct {
	GroupID    string         `json:"group_id"`
	MessageID  string         `json:"message_id"`
	RecalledBy RecalledByUser `json:"recalled_by"`
	RecalledAt string         `json:"recalled_at"`
}

type GroupMessageReadCountUpdatedPayload struct {
	GroupID   string `json:"group_id"`
	MessageID string `json:"message_id"`
	ReadCount int    `json:"read_count"`
	UpdatedAt string `json:"updated_at"`
}
