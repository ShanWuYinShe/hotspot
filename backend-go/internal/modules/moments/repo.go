package moments

import "context"

type Repository interface {
	Create(context.Context, string, CreateMomentRequest) MomentItem
	Get(context.Context, string) (MomentItem, bool)
	ListMine(context.Context, string) PaginatedMoments
	IncrementCommentCount(context.Context, string)
}
