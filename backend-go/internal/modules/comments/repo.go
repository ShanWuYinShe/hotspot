package comments

import "context"

type Repository interface {
	Create(context.Context, string, CreateCommentRequest) (CommentItem, error)
	// SoftDelete 原子软删并返回是否实际命中（并发重复删除时第二次返回
	// false，调用方据此跳过计数递减——检查-使用窗口的双重递减防线）。
	SoftDelete(context.Context, string) (bool, error)
	List(context.Context, string, int, string) (PaginatedComments, error)
	Get(ctx context.Context, commentID string) (CommentItem, bool, error)
	ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error)
	ListTopRepliesByParentIDs(ctx context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error)
}
