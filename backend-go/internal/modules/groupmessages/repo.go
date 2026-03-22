package groupmessages

import "context"

type Repository interface {
	Send(context.Context, SendGroupMessageRequest) (MessageSent, error)
	History(context.Context, string) (PaginatedGroupMessages, error)
}
