package privatemessages

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMessageNotFound         = errors.New("private_messages: message not found")
	ErrMessagePermissionDenied = errors.New("private_messages: permission denied")
	ErrMessageNotEditable      = errors.New("private_messages: message not editable")
	ErrMessageNotRecallable    = errors.New("private_messages: message not recallable")
	ErrInvalidCursor           = errors.New("private_messages: invalid cursor")
	// ErrInvalidShareCode：share 类型消息的 share_code 无效（不存在/已撤销/已过期）。
	ErrInvalidShareCode = errors.New("private_messages: invalid share code")
)

// BlockReason mirrors spec api_spec_chat.md BlockReason: the structured reason
// returned in the 10200 error data of private_messages/send.
// 拉黑明确返回 'blocked'（产品决策：拉黑就显示拉黑，见 spec）；
// 'privacy' 仅用于 allow_private_chat 隐私设置限制。
type BlockReason string

const (
	// BlockReasonBlocked: the recipient has blocked the sender.
	BlockReasonBlocked BlockReason = "blocked"
	// BlockReasonPrivacy: privacy settings (allow_private_chat) disallow the message.
	BlockReasonPrivacy BlockReason = "privacy"
	// BlockReasonStrangerQuotaExceeded: daily stranger-message quota used up.
	BlockReasonStrangerQuotaExceeded BlockReason = "stranger_quota_exceeded"
)

// MessageBlockedError is returned by Service.Send when the message is refused
// with a structured BlockReason (HTTP 200 + code 10200 + data {reason}).
type MessageBlockedError struct {
	Reason BlockReason
}

func (e *MessageBlockedError) Error() string {
	return "private_messages: send blocked: " + string(e.Reason)
}

type MessageEventMeta struct {
	TargetUserID string
	MessageID    string
	SentAt       string
	Content      string
}

type Repository interface {
	Send(context.Context, string, string, SendPrivateMessageRequest) (MessageSent, error)
	History(context.Context, string, GetHistoryRequest) (PaginatedMessages, error)
	Recent(context.Context, string, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, string, *time.Time) (int, error)
	Update(context.Context, string, UpdatePrivateMessageRequest) (MessageEventMeta, error)
	Recall(context.Context, string, RecallPrivateMessageRequest) (MessageEventMeta, error)
	MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) (MessageEventMeta, error)
	MarkAllRead(context.Context, string) error
	GetMessageForReply(context.Context, string, string) (map[string]any, bool, error)
	GetMediaForMessage(context.Context, string, string) (map[string]any, bool, error)
	MessageExists(ctx context.Context, messageID string) (found bool, senderID string, err error)
	// CountMessagesByTarget counts messages sent by senderID since the given
	// time, grouped by the receiving user (used for the stranger daily quota
	// check which is total-volume based across all non-mutual targets).
	CountMessagesByTarget(ctx context.Context, senderID string, since time.Time) (map[string]int, error)
}
