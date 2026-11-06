package moments

import "context"

type Repository interface {
	Create(context.Context, string, CreateMomentRequest) (MomentItem, error)
	Get(context.Context, string) (MomentItem, bool, error)
	// Search 全球 keyword/tag 搜索。viewerID 为登录查看者（匿名传空串）：非空时
	// 结果需填充查看者 is_favorited（spec api_spec_types.md MomentItem 查看者布尔）。
	Search(context.Context, SearchMomentsRequest, string) (PaginatedMoments, error)
	Update(context.Context, string, UpdateMomentRequest) error
	Delete(context.Context, string) error
	ListMine(context.Context, string, int, string) (PaginatedMoments, error)
	// AdjustInteractionCount 维护内部互动热度列 interaction_count（delta 可负，
	// 下限钳 0）：供 reactions/comments/share 模块在写成功后调用，只服务
	// nearby/popular 排序（契约计数已改读时聚合，不依赖本列）。
	AdjustInteractionCount(context.Context, string, int) error
	Favorite(context.Context, string, string) error
	Unfavorite(context.Context, string, string) error
	ListFavorites(context.Context, string, int, string) (PaginatedMoments, error)
	// ListFavoriteUserIDs returns the user IDs who favorited the given moment
	// (used to fan out MomentDeleted to favoriters, spec api_spec_websocket.md).
	// ListFavoriteUserIDs 返回收藏该瞬间的 user_id（最多 limit 条；limit<=0 表示不限制）。
	// 删除瞬间的收藏者广播属尽力而为，由调用方传入上限，避免热门瞬间全量载入内存
	// （审查 P2-17：此前 repo 无 LIMIT，service 截断前已把全部行读进内存）。
	ListFavoriteUserIDs(ctx context.Context, momentID string, limit int) ([]string, error)
	LoadMyState(context.Context, string, string) *MyMomentState
	// FetchMomentImageRows loads ordered image rows for the given moment IDs
	// (used by the service-layer image hydration helper).
	FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error)
	// ListFollowingMoments 关注动态（spec moment.md → moments/fetch_following，
	// 原 feed/following 迁移）：created_at DESC, moment_id DESC keyset。
	ListFollowingMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	// ListNearbyMoments 附近瞬间（spec moment.md → moments/fetch_nearby，
	// 原 feed/nearby_moments 迁移）：distance 主序。
	ListNearbyMoments(context.Context, momentsQuery) (PaginatedMoments, error)
	// ListMomentsByAuthor / ListReactedByUser 供 users 模块个人主页聚合消费
	// （原实现位于 users repo，直查 moments/reactions 表；现归位本模块，
	// users 经 Service 接口消费，见 Service.ListMomentsByAuthor）。
	ListMomentsByAuthor(context.Context, string, int, string, bool) (PaginatedMoments, error)
	ListReactedByUser(context.Context, string, string, int, string) (PaginatedMoments, error)
}
