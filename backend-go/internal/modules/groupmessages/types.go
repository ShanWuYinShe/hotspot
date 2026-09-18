package groupmessages

import (
	"context"
	"time"

	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

// MessageBrief 是消息摘要快照（reply_to 回复引用；spec api_spec_types.md——
// 与私聊侧 privatemessages.MessageBrief 同构）。
type MessageBrief struct {
	MessageID string         `json:"message_id"`
	Type      string         `json:"type"`
	Content   *string        `json:"content"`
	ShareCode *string        `json:"share_code"`
	Sender    types.UserInfo `json:"sender"`
	SentAt    string         `json:"sent_at"`
}

// Reaction 是消息表情反应聚合（spec api_spec_chat.md Reaction：空数组 = 无
// 反应；reacted_by_me 恒定布尔）。本地声明避免模块间 import 依赖，由
// bootstrap 适配 reactions 模块的同名类型。
type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	ReactedByMe bool   `json:"reacted_by_me"`
}

// MessageReactionsSource 提供消息级表情反应批量聚合（bootstrap 注入 reactions
// 模块适配器）。nil 时 reactions 恒为空数组（降级不报错）。
type MessageReactionsSource interface {
	AggregatedGroupMessageReactions(ctx context.Context, messageIDs []string) (map[string][]Reaction, error)
}

type GroupMessageItem struct {
	MessageID    string         `json:"message_id"`
	SenderUserID string         `json:"-"` // 内部存储用；对外通过 Sender map 暴露
	Type         string         `json:"type"`
	Content      *string        `json:"content"`
	Media        map[string]any `json:"media"`
	ShareCode    *string        `json:"share_code"`
	Sender       map[string]any `json:"sender"` // UserInfo
	ReplyTo      *MessageBrief  `json:"reply_to"`
	Mentions     []string       `json:"mentions"`
	RecalledBy   map[string]any `json:"recalled_by"`
	SentAt       string         `json:"sent_at"`
	EditedAt     *string        `json:"edited_at"`
	RecalledAt   *string        `json:"recalled_at"`
	ReadCount    *int           `json:"read_count"`
	Reactions    []Reaction     `json:"reactions"`
}
type MessageSentResult struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
type SendGroupMessageRequest struct {
	GroupID          string   `json:"group_id"`
	SenderUserID     string   `json:"-"`
	Type             string   `json:"type"`
	Content          string   `json:"content,omitempty"`
	FileID           string   `json:"file_id,omitempty"`
	ShareCode        string   `json:"share_code,omitempty"`
	ReplyToMessageID string   `json:"reply_to_message_id,omitempty"`
	Mentions         []string `json:"mentions,omitempty"`
}
type PaginationParams = types.PaginationParams

type groupMessageCursor struct {
	SentAt    string `json:"sent_at"`
	MessageID string `json:"message_id"`
}

func encodeGroupMessageCursor(sentAt, messageID string) string {
	value, err := platformcursor.Encode("group_messages.history", groupMessageCursor{SentAt: sentAt, MessageID: messageID})
	if err != nil {
		return ""
	}
	return value
}

func decodeGroupMessageCursor(value string) (time.Time, string, error) {
	if value == "" {
		return time.Time{}, "", nil
	}
	var decoded groupMessageCursor
	if err := platformcursor.Decode("group_messages.history", value, &decoded); err != nil || decoded.SentAt == "" || decoded.MessageID == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.SentAt)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.MessageID, nil
}

type ListGroupMessagesRequest struct {
	GroupID    string            `json:"group_id"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}
type PaginatedGroupMessages struct {
	Items      []GroupMessageItem `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

// 新增请求类型
type UpdateGroupMessageRequest struct {
	MessageID string `json:"message_id"`
	UserID    string `json:"-"`
	Content   string `json:"content"`
}

type RecallGroupMessageRequest struct {
	MessageID string `json:"message_id"`
	UserID    string `json:"-"`
}

type MarkGroupMessagesReadRequest struct {
	GroupID       string `json:"group_id"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

// MarkGroupReadRequest is the unified mark_read request for group messages.
// scope defaults to "group"; "group" requires group_id (optionally
// up_to_message_id), "all" clears all unread group messages for the caller.
type MarkGroupReadRequest struct {
	Scope         string `json:"scope,omitempty"`
	GroupID       string `json:"group_id,omitempty"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

// MessageReportSnapshot 是群消息的举报快照投影视图（spec api_spec_types.md
// ReportSnapshot 的 group 会话形态）：服务端受理举报时基于消息真值生成，
// 由 bootstrap 适配为 reports.ReportSnapshot（groupmessages 不依赖 reports 包）。
type MessageReportSnapshot struct {
	GroupID string // target_chat_id（group 会话 = group_id）
	Type    string // message_type
	Content string // 原始文本内容（type='text' 时作为 snapshot_content）
	FileID  string // 媒体文件引用（非空时作为 snapshot_media.file_id）
	SentAt  string
}
