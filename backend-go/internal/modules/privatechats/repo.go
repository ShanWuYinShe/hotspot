package privatechats

import (
	"context"
	"time"
)

type ChatState struct {
	ChatKey      string
	TargetID     string
	CreatedAt    time.Time
	IsMuted      bool
	IsPinned     bool
	VisibleSince *time.Time
	DeletedAt    *time.Time
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
