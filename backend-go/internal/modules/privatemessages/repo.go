package private_messages

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
)

// BlockReason mirrors spec api_spec_chat.md BlockReason: the structured reason
// returned in the 10200 error data of private_messages/send.
type BlockReason string

const (
	// BlockReasonPrivacy covers the blocked relationship and privacy settings
	// (allow_private_chat) — the blocked state is never exposed.
	BlockReasonPrivacy BlockReason = "privacy"
	// BlockReasonStrangerQuotaExceeded: daily stranger-message quota used up.
	BlockReasonStrangerQuotaExceeded BlockReason = "stranger_quota_exceeded"
	// BlockReasonAccountSuspended: reserved (account restriction) — type kept
	// for contract parity; not enforced yet.
	BlockReasonAccountSuspended BlockReason = "account_suspended"
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
	// CountMessagesToUser counts messages sent by senderID to targetUserID
	// since the given time (used for the stranger daily quota check).
	CountMessagesToUser(ctx context.Context, senderID, targetUserID string, since time.Time) (int, error)
}
