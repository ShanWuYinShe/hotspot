package moments

import "context"

type Repository interface {
	Create(context.Context, string, CreateMomentRequest) (MomentItem, error)
	Get(context.Context, string) (MomentItem, bool, error)
	Search(context.Context, SearchMomentsRequest) (PaginatedMoments, error)
	Update(context.Context, string, EditMomentRequest) error
	Delete(context.Context, string) error
	ListMine(context.Context, string) (PaginatedMoments, error)
	Exists(context.Context, string) (bool, error)
	IncrementCommentCount(context.Context, string) error
}
