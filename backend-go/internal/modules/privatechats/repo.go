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
}
