package groupmessages

import (
	"time"

	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

// MessageBrief 是消息摘要快照（reply_to 回复引用；spec api_spec_types.md——
// 与私聊侧 privatemessages.MessageBrief 同构）。
type MessageBrief struct {
	MessageID string         `json:"message_id"`
	Type      string         `json:"type"`
	Content   string         `json:"content,omitempty"`
	ShareCode string         `json:"share_code,omitempty"`
	Sender    types.UserInfo `json:"sender"`
	SentAt    string         `json:"sent_at"`
}

type GroupMessageItem struct {
	MessageID    string         `json:"message_id"`
	SenderUserID string         `json:"-"` // 内部存储用；对外通过 Sender map 暴露
	Type         string         `json:"type"`
	Content      string         `json:"content,omitempty"`
	Media        map[string]any `json:"media,omitempty"`
	ShareCode    string         `json:"share_code,omitempty"`
	Sender       map[string]any `json:"sender"` // UserInfo
	ReplyTo      *MessageBrief `json:"reply_to,omitempty"`
	Mentions     []string       `json:"mentions,omitempty"`
	RecalledBy   map[string]any `json:"recalled_by,omitempty"`
	SentAt       string         `json:"sent_at"`
	EditedAt     string         `json:"edited_at,omitempty"`
	RecalledAt   string         `json:"recalled_at,omitempty"`
	ReadCount    int            `json:"read_count,omitempty"`
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
	NextCursor string             `json:"next_cursor,omitempty"`
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
