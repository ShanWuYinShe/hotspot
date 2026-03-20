package groupmessages

import "context"

type Repository interface {
	Send(context.Context, SendGroupMessageRequest) MessageSent
	History(context.Context, string) PaginatedGroupMessages
}
