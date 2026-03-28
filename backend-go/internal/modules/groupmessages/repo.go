package groupmessages

import "context"

type Repository interface {
	Send(context.Context, SendGroupMessageRequest) (MessageSent, error)
	History(context.Context, string) (PaginatedGroupMessages, error)

	// 新增方法
	Update(context.Context, UpdateGroupMessageRequest) error
	Recall(context.Context, RecallGroupMessageRequest) error
	MarkAsRead(context.Context, string, string) error  // groupID, upToMessageID
	MarkAllRead(context.Context) error
}
