package moments

import "context"

type Repository interface {
	Create(context.Context, string, CreateMomentRequest) (MomentItem, error)
	Get(context.Context, string) (MomentItem, bool, error)
	ListMine(context.Context, string) (PaginatedMoments, error)
	IncrementCommentCount(context.Context, string) error
}
