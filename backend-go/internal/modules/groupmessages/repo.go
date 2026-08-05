package groupmessages

import "context"

type MessageEventMeta struct {
	GroupID   string
	MessageID string
	SentAt    string
	Content   string
}

// MessageOwnerMeta carries the ownership metadata needed to authorize
// Update/Recall: which group the message lives in and who sent it.
// SenderUserID is "" for messages with a NULL sender (legacy/anonymous);
// such messages are not editable/recallable by anyone.
type MessageOwnerMeta struct {
	GroupID      string
	SenderUserID string
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
	MessageExists(ctx context.Context, messageID string) (found bool, senderID string, err error)
	// GetGroupName returns the group name for the given group (used in
	// GroupMessageMentioned notification payloads).
	GetGroupName(ctx context.Context, groupID string) (string, bool, error)
	// LookupMessageMeta returns the group ID and sender user ID for a message.
	// senderID is "" when the message has a NULL sender (legacy/anonymous).
	// found is false when no message matches messageID.
	LookupMessageMeta(ctx context.Context, messageID string) (meta MessageOwnerMeta, found bool, err error)
}
