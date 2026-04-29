package moments

import "context"

type Repository interface {
	Create(context.Context, string, CreateMomentRequest) (MomentItem, error)
	Get(context.Context, string) (MomentItem, bool, error)
	Search(context.Context, SearchMomentsRequest) (PaginatedMoments, error)
	Update(context.Context, string, EditMomentRequest) error
	Delete(context.Context, string) error
	ListMine(context.Context, string, int, string) (PaginatedMoments, error)
	Exists(context.Context, string) (bool, error)
	IncrementCommentCount(context.Context, string) error
	DecrementCommentCount(context.Context, string) error
	IncrementSharesCount(context.Context, string) error
	Like(context.Context, string, string) (bool, error)
	Unlike(context.Context, string, string) error
	Favorite(context.Context, string, string) error
	Unfavorite(context.Context, string, string) error
	ListLikes(context.Context, string, int, string) ([]LikeItem, string, error)
	ListFavorites(context.Context, string, int, string) (PaginatedMoments, error)
	LoadMyState(context.Context, string, string) *MyMomentState
}
