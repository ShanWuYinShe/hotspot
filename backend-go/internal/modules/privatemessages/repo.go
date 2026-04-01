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
)

type Repository interface {
	Send(context.Context, string, string, SendPrivateMessageRequest) (MessageSent, error)
	History(context.Context, string, string) (PaginatedMessages, error)
	Recent(context.Context, string, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, string, *time.Time) (int, error)
	Update(context.Context, string, UpdatePrivateMessageRequest) error
	Recall(context.Context, string, RecallPrivateMessageRequest) error
	MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) error
	MarkAllRead(context.Context, string) error
}
