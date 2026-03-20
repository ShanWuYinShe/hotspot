package comments

import "context"

type Repository interface {
	Create(context.Context, string, CreateCommentRequest) CommentItem
	List(context.Context, string) PaginatedComments
}
