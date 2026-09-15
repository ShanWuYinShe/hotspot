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
	// ErrFileNotOwned：媒体消息的 file_id 不存在、不属于发送者或未完成上传
	// （对齐 moments 图片归属口径，spec 10300）。
	ErrFileNotOwned = errors.New("private_messages: file not owned by sender")
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
	// LastByChatKeys / UnreadCountsByChatKeys：会话列表批量读（privatechats.List
	// 消费，替代逐会话 Last/UnreadCount 的 N+1；boundaries 缺失的 key 视为 nil）。
	LastByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]PrivateMessageItem, error)
	UnreadCountsByChatKeys(ctx context.Context, userID string, chatKeys []string, boundaries map[string]*time.Time) (map[string]int, error)
	Update(context.Context, string, UpdatePrivateMessageRequest) (MessageEventMeta, error)
	// Recall 撤回消息。window 为撤回时间窗（AppConfig.limits.
	// message_recall_window_seconds，由 service 从 appConfig 取值传入）；
	// window<=0 表示撤回禁用（spec chat.md：0=禁用），一律拒绝。
	Recall(ctx context.Context, userID string, window time.Duration, req RecallPrivateMessageRequest) (MessageEventMeta, error)
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
	// LockStrangerQuota 对「发送者 + dayKey(UTC 日期)」取事务级咨询锁（PG
	// advisory xact lock），串行化同发送者的陌生人配额检查与消息写入，消除
	// check-then-act 竞态。必须在 RunInTransaction 事务内、配额计数之前调用；
	// 非事务 ctx 或内存实现为 no-op。
	LockStrangerQuota(ctx context.Context, senderID, dayKey string) error
}
