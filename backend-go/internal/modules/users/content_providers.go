package users

import (
	"context"
	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
)

// content_providers.go —— 个人主页聚合（fetch_groups / fetch_moments /
// list_reacted）的跨模块读依赖。users 只依赖下列窄接口；实现由 groups/moments
// 的 Service 提供、bootstrap 注入，替代原先 users repo 直查 groups/moments/
// reactions 表的跨模块 SQL（AGENTS.md：模块间经 Service 接口解耦）。

// GroupsByUserProvider 返回某用户的可见群组（searchable、未删除），
// joined_at DESC keyset。
type GroupsByUserProvider interface {
	ListGroupsByUser(ctx context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error)
	// CountCommonGroups 两用户共同群组数（主页「共同群组」徽标）。
	CountCommonGroups(ctx context.Context, viewerUserID, targetUserID string) (int, error)
}

// MomentsByUserProvider 返回按作者聚合的瞬间列表。
type MomentsByUserProvider interface {
	// ListMomentsByAuthor 非删除瞬间，created_at DESC keyset；
	// includePrivate=false 限定公开（公开主页视角）。
	ListMomentsByAuthor(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error)
	// ListReactedByUser 某用户反应过的瞬间，reacted_at DESC keyset；
	// viewerUserID 非空时解锁该作者自己的私密瞬间。
	ListReactedByUser(ctx context.Context, userID, viewerUserID string, limit int, cursor string) (momenttypes.PaginatedMoments, error)
}
