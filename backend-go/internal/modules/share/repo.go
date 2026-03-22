package share

import "context"

type Repository interface {
	Create(context.Context, string, CreateShareRequest) (ShareCreated, error)
	Get(context.Context, string) (ShareDetail, bool, error)
	List(context.Context, string) (PaginatedShares, error)
	Revoke(context.Context, string, string) (bool, error)
}
