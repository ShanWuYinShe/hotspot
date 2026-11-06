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
	CheckCommentable(context.Context, string) (found bool, canComment bool, err error)
	// IsReactionTargetVisible 判定瞬间对 viewer 可见（软删/private 统一不可见）。
	// 评论读路径（List/open_the_door）复用它做可见性校验，防止知道 moment_id
	// 的任意用户读取 private 瞬间的全部评论。
	IsReactionTargetVisible(context.Context, string) (found bool, visible bool, err error)
	// AdjustInteractionCount 维护瞬间内部互动热度（delta 可负）；失败由 moments
	// 侧记录 WARN，不影响评论主流程（热度只用于排序，契约计数读时聚合）。
	AdjustInteractionCount(context.Context, string, int)
	NotificationContext(context.Context, string) (MomentNotificationContext, bool, error)
}
