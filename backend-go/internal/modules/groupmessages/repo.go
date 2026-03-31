package groupmessages

import "context"

type Repository interface {
	Send(context.Context, SendGroupMessageRequest) (MessageSent, error)
	History(context.Context, string) (PaginatedGroupMessages, error)

	Update(context.Context, UpdateGroupMessageRequest) error
	Recall(context.Context, RecallGroupMessageRequest) error
	MarkAsRead(context.Context, string, MarkGroupMessagesReadRequest) error
	MarkAllRead(context.Context, string) error
}
