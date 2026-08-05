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
	Favorite(context.Context, string, string) error
	Unfavorite(context.Context, string, string) error
	ListFavorites(context.Context, string, int, string) (PaginatedMoments, error)
	// ListFavoriteUserIDs returns the user IDs who favorited the given moment
	// (used to fan out MomentDeleted to favoriters, spec api_spec_websocket.md).
	ListFavoriteUserIDs(context.Context, string) ([]string, error)
	LoadMyState(context.Context, string, string) *MyMomentState
	// FetchMomentImageRows loads ordered image rows for the given moment IDs
	// (used by the service-layer image hydration helper).
	FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error)
}
