package comments

import "context"

type MomentProvider interface {
	Exists(context.Context, string) (bool, error)
	IncrementCommentCount(context.Context, string) error
	DecrementCommentCount(context.Context, string) error
}
