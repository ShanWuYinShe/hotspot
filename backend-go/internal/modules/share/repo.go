package share

import "context"

type Repository interface {
	Create(context.Context, string, CreateShareRequest) ShareCreated
	Get(context.Context, string) (ShareDetail, bool)
	List(context.Context, string) PaginatedShares
	Revoke(context.Context, string, string) bool
}
