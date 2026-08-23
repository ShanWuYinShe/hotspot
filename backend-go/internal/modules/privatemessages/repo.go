package privatemessages

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
	ErrInvalidCursor           = errors.New("private_messages: invalid cursor")
	// ErrInvalidShareCode：share 类型消息的 share_code 无效（不存在/已撤销/已过期）。
	ErrInvalidShareCode = errors.New("private_messages: invalid share code")
)

type MessageEventMeta struct {
	TargetUserID string
	MessageID    string
	SentAt       string
	Content      string
}

type Repository interface {
	Send(context.Context, string, string, SendPrivateMessageRequest) (MessageSentResult, error)
	History(context.Context, string, ListPrivateMessagesRequest) (PaginatedMessages, error)
	Recent(context.Context, string, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, string, *time.Time) (int, error)
	Update(context.Context, string, UpdatePrivateMessageRequest) (MessageEventMeta, error)
	Recall(context.Context, string, RecallPrivateMessageRequest) (MessageEventMeta, error)
	MarkAsRead(context.Context, string, MarkPrivateMessagesReadRequest) (MessageEventMeta, error)
	MarkAllRead(context.Context, string) error
	GetMessageForReply(context.Context, string, string) (map[string]any, bool, error)
	GetMediaForMessage(context.Context, string, string) (map[string]any, bool, error)
	MessageExists(ctx context.Context, messageID string) (found bool, senderID string, err error)
	MessageParticipants(ctx context.Context, messageID string) (senderID, receiverID string, found bool, err error)
	// GetMessageReportSnapshot 按 message_id 读取消息真值生成举报快照投影视图
	//（spec api_spec_types.md ReportSnapshot 的 private 会话形态，由 bootstrap
	// 适配为 reports.ReportSnapshot；举报快照由服务端生成，客户端不提交）。
	GetMessageReportSnapshot(ctx context.Context, messageID string) (snapshot MessageReportSnapshot, found bool, err error)
	// CountMessagesByTarget counts messages sent by senderID since the given
	// time, grouped by the receiving user (used for the stranger daily quota
	// check which is total-volume based across all non-mutual targets).
	CountMessagesByTarget(ctx context.Context, senderID string, since time.Time) (map[string]int, error)
}
