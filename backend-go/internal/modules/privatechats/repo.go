package privatechats

import (
	"context"
	"time"
)

type ChatState struct {
	ChatKey   string
	TargetID  string
	CreatedAt time.Time
	// LastActivityAt 最近可见消息时间（无消息时回退为 CreatedAt）。列表排序
	// 键，分页下推后由 SQL 的 LATERAL 计算并返回（审查 P2-7）。
	LastActivityAt time.Time
	IsMuted        bool
	IsPinned       bool
	VisibleSince   *time.Time
	DeletedAt      *time.Time
}

type UpdateChatSettings struct {
	IsMuted  *bool
	IsPinned *bool
}

type ListOptions struct {
	Limit  int
	Cursor string
}

type Repository interface {
	GetOrCreate(context.Context, string, string) (time.Time, error)
	List(context.Context, string, ListOptions) ([]ChatState, string, error)
	GetState(context.Context, string, string) (ChatState, error)
	RestoreForUser(context.Context, string, string, time.Time) error
	UpdateSettings(context.Context, string, string, UpdateChatSettings) error
	DeleteForUser(context.Context, string, string, *time.Time) error
	// CountUnreadForUser 统计该用户全部私聊会话的未读消息总数（冷启动
	// unread_summary 聚合消费）。口径与列表逐会话 unread_count 一致：以对方
	// 视角的已读游标（private_chat_read_state.last_read_sent_at）为界，叠加
	// 会话可见边界（private_chat_user_state.visible_since/deleted_at），并排除
	// 本人发送与已撤回消息；对方已注销的会话不计（列表同样不可达）。
	CountUnreadForUser(context.Context, string) (int, error)
}
