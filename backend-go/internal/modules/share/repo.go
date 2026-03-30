package share

import "context"

type Repository interface {
	Create(context.Context, createShareCommand) (ShareCreated, error)
	Get(context.Context, getShareQuery) (ShareDetail, error)
	List(context.Context, listSharesQuery) (PaginatedShares, error)
	Revoke(context.Context, revokeShareCommand) error
}
