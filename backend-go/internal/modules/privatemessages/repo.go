package private_messages

import (
	"context"
	"errors"
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
	Recent(context.Context, string, string, int) (PaginatedMessages, error)
	Last(context.Context, string, string) (*PrivateMessageItem, error)
	Update(context.Context, string, UpdatePrivateMessageRequest) error
	Recall(context.Context, string, RecallPrivateMessageRequest) error
	MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) error
	MarkAllRead(context.Context, string) error
}
