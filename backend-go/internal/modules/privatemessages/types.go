package privatemessages

import (
	"time"

	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

type PaginationParams = types.PaginationParams

type messageCursor struct {
	SentAt    string `json:"sent_at"`
	MessageID string `json:"message_id"`
}

func encodeMessageCursor(sentAt, messageID string) string {
	value, err := platformcursor.Encode("private_messages.history", messageCursor{SentAt: sentAt, MessageID: messageID})
	if err != nil {
		return ""
	}
	return value
}

func decodeMessageCursor(value string) (time.Time, string, error) {
	if value == "" {
		return time.Time{}, "", nil
	}
	var decoded messageCursor
	if err := platformcursor.Decode("private_messages.history", value, &decoded); err != nil || decoded.SentAt == "" || decoded.MessageID == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.SentAt)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.MessageID, nil
}

type UserInfo = types.UserInfo

type MessageMedia struct {
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	FileName     *string `json:"file_name"`
	FileSize     *int    `json:"file_size"`
	Duration     *int    `json:"duration"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	MimeType     *string `json:"mime_type"`
}

type MessageBrief struct {
	MessageID string   `json:"message_id"`
	Type      string   `json:"type"`
	Content   *string  `json:"content"`
	ShareCode *string  `json:"share_code"`
	Sender    UserInfo `json:"sender"`
	SentAt    string   `json:"sent_at"`
}

type PrivateMessageItem struct {
	MessageID  string        `json:"message_id"`
	Type       string        `json:"type"`
	Content    *string       `json:"content"`
	Media      *MessageMedia `json:"media"`
	ShareCode  *string       `json:"share_code"`
	Sender     UserInfo      `json:"sender"`
	ReplyTo    *MessageBrief `json:"reply_to"`
	SentAt     string        `json:"sent_at"`
	EditedAt   *string       `json:"edited_at"`
	RecalledAt *string       `json:"recalled_at"`
	IsRead     bool          `json:"is_read"`
}

type PaginatedMessages struct {
	Items      []PrivateMessageItem `json:"items"`
	NextCursor *string              `json:"next_cursor"`
}

// MessageReportSnapshot 是私聊消息的举报快照投影视图（spec api_spec_types.md
// ReportSnapshot 的 private 会话形态）：服务端受理举报时基于消息真值生成，
// 由 bootstrap 适配为 reports.ReportSnapshot（privatemessages 不依赖 reports 包）。
// TargetUserID 为会话对方（相对发送者的另一方，spec：private 会话 = 对方 user_id）。
type MessageReportSnapshot struct {
	TargetUserID string
	Type         string
	Content      string // 原始文本内容（type='text' 时作为 snapshot_content）
	FileID       string // 媒体文件引用（非空时作为 snapshot_media.file_id）
	SentAt       string
}

type SendPrivateMessageRequest struct {
	TargetUserID     string `json:"target_user_id"`
	Type             string `json:"type"`
	Content          string `json:"content,omitempty"`
	FileID           string `json:"file_id,omitempty"`
	ShareCode        string `json:"share_code,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
}

type ListPrivateMessagesRequest struct {
	TargetUserID string            `json:"target_user_id"`
	Pagination   *PaginationParams `json:"pagination,omitempty"`
}

type UpdatePrivateMessageRequest struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type RecallPrivateMessageRequest struct {
	MessageID string `json:"message_id"`
}

type MarkPrivateMessagesReadRequest struct {
	TargetUserID  string `json:"target_user_id"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

// MarkPrivateReadRequest is the unified mark_read request for private messages.
// scope defaults to "session"; "session" requires target_user_id (optionally
// up_to_message_id), "all" clears all unread private messages for the caller.
type MarkPrivateReadRequest struct {
	Scope         string `json:"scope,omitempty"`
	TargetUserID  string `json:"target_user_id,omitempty"`
	UpToMessageID string `json:"up_to_message_id,omitempty"`
}

type MessageSentResult struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
