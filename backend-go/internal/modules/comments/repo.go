package comments

import "context"

type Repository interface {
	Create(context.Context, string, CreateCommentRequest) (CommentItem, error)
	Delete(context.Context, string) error
	List(context.Context, string) (PaginatedComments, error)
}
