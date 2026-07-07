package groupmessages

import "context"

type MessageEventMeta struct {
	GroupID   string
	MessageID string
	SentAt    string
	Content   string
}

type Repository interface {
	Send(context.Context, SendGroupMessageRequest) (MessageSent, error)
	History(context.Context, string, *PaginationParams) (PaginatedGroupMessages, error)

	Update(context.Context, UpdateGroupMessageRequest) (MessageEventMeta, error)
	Recall(context.Context, RecallGroupMessageRequest) (MessageEventMeta, error)
	MarkAsRead(context.Context, string, MarkGroupMessagesReadRequest) error
	MarkAllRead(context.Context, string) error
	IsMember(context.Context, string, string) (bool, error)
	GetMessageForReply(context.Context, string) (map[string]any, bool, error)
	GetMediaForMessage(context.Context, string) (map[string]any, bool, error)
	MemberMuteStatus(context.Context, string, string) (isMuted bool, mutedUntil string, found bool, err error)
}
