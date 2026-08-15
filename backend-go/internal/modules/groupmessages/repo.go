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
	Send(context.Context, SendGroupMessageRequest) (MessageSentResult, error)
	History(context.Context, string, *PaginationParams) (PaginatedGroupMessages, error)

	Update(context.Context, UpdateGroupMessageRequest) (MessageEventMeta, error)
	Recall(context.Context, RecallGroupMessageRequest) (MessageEventMeta, error)
	// RecallAsAdmin recalls a group message regardless of who sent it, with no
	// recall-window enforcement. Used by the groups module's admin guard
	// endpoint groups/admin/messages/recall (spec api_spec_group.md). Only
	// not-found (ErrMessageNotFound) and already-recalled (ErrMessageRecalled)
	// conflicts are returned.
	RecallAsAdmin(ctx context.Context, messageID string) (MessageEventMeta, error)
	MarkAsRead(context.Context, string, MarkGroupMessagesReadRequest) error
	// MarkAllRead 标记用户全部群消息已读，返回受影响的 group_ids（驱动
	// GroupMessageReadCountUpdated 的节流推送）。
	MarkAllRead(context.Context, string) ([]string, error)
	// GetReadCount 返回群消息当前已读成员数（read_count 事件负载）。
	GetReadCount(ctx context.Context, groupID string, messageID string) (int, error)
	// LatestMessageID 返回群内最新消息 ID（mark_read 无 up_to 时的事件目标）。
	LatestMessageID(ctx context.Context, groupID string) (string, error)
	IsMember(context.Context, string, string) (bool, error)
	// FilterMemberIDs 批量过滤出属于该群成员的 user_ids（mentions 通知只对
	// 真实成员落库/推送，防内容泄漏与通知轰炸）。
	FilterMemberIDs(ctx context.Context, groupID string, userIDs []string) ([]string, error)
	GetMessageForReply(context.Context, string) (*MessageBrief, bool, error)
	GetMediaForMessage(context.Context, string) (map[string]any, bool, error)
	MemberMuteStatus(context.Context, string, string) (isMuted bool, mutedUntil string, found bool, err error)
	MessageExists(ctx context.Context, messageID string) (found bool, senderID string, err error)
	MessageGroupID(ctx context.Context, messageID string) (groupID string, found bool, err error)
	// GetGroupName returns the group name for the given group (used in
	// GroupMessageMentioned notification payloads).
	GetGroupName(ctx context.Context, groupID string) (string, bool, error)
	// LookupMessageMeta returns the group ID and sender user ID for a message.
	// senderID is "" when the message has a NULL sender (legacy/anonymous).
	// found is false when no message matches messageID.
	LookupMessageMeta(ctx context.Context, messageID string) (meta MessageOwnerMeta, found bool, err error)
}
