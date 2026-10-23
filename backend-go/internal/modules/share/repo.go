package share

import "context"

type Repository interface {
	Create(context.Context, createShareCommand) (ShareCreated, error)
	Get(context.Context, getShareQuery) (ShareDetail, error)
	// ResolveShareIDByCode 按分享码解析 share_id，**不过滤 revoked/expired**。
	// 撤销幂等需要：按 code 重复撤销（或撤销已过期分享）应幂等成功而非 404，
	// 而 Get 会把不可用状态统一映射 ErrShareNotFound（审查 P2-8 幂等分裂）。
	// 返回 (…, false, nil) 表示分享码不存在。
	ResolveShareIDByCode(ctx context.Context, shareCode string) (string, bool, error)
	List(context.Context, listSharesQuery) (PaginatedShares, error)
	Revoke(context.Context, revokeShareCommand) error
}
