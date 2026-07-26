package comments

import "context"

type Repository interface {
	Create(context.Context, string, CreateCommentRequest) (CommentItem, error)
	SoftDelete(context.Context, string) error
	List(context.Context, string, int, string) (PaginatedComments, error)
	Get(ctx context.Context, commentID string) (CommentItem, bool, error)
	ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error)
	ListTopRepliesByParentIDs(ctx context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error)
}
