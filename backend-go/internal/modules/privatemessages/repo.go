package private_messages

import "context"

type Repository interface {
	Send(context.Context, string, string, SendPrivateMessageRequest) MessageSent
	History(context.Context, string, string) PaginatedMessages
	Recent(context.Context, string, string, int) PaginatedMessages
	Last(context.Context, string, string) *PrivateMessageItem
	Update(context.Context, string, UpdatePrivateMessageRequest) error
	Recall(context.Context, string, RecallPrivateMessageRequest) error
	MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) error
	MarkAllRead(context.Context, string) error
}
