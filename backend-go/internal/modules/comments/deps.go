package comments

import (
	"context"

	"hotspot/backend-go/internal/platform/types"
)

type MomentNotificationContext struct {
	UserID  string
	Summary types.MomentSummary
}

type MomentProvider interface {
	Exists(context.Context, string) (bool, error)
	CheckCommentable(context.Context, string) (found bool, canComment bool, err error)
	IncrementCommentCount(context.Context, string) error
	DecrementCommentCount(context.Context, string) error
	NotificationContext(context.Context, string) (MomentNotificationContext, bool, error)
}
