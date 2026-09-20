package groupmessages

import (
	"hotspot/backend-go/internal/platform/types"
)

// ws_payloads.go — groupmessages 模块的 WebSocket 出站 payload 类型（spec
// api_spec_websocket.md 实现侧镜像）。struct 化后字段漂移在编译期暴露。

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
