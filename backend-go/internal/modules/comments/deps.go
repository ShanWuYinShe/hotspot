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
	// IsReactionTargetVisible 判定瞬间对 viewer 可见（软删/private 统一不可见）。
	// 评论读路径（List/open_the_door）复用它做可见性校验，防止知道 moment_id
	// 的任意用户读取 private 瞬间的全部评论。
	IsReactionTargetVisible(context.Context, string) (found bool, visible bool, err error)
	IncrementCommentCount(context.Context, string) error
	DecrementCommentCount(context.Context, string) error
	NotificationContext(context.Context, string) (MomentNotificationContext, bool, error)
}
