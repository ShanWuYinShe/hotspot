package comments

import "context"

type MomentCounter interface {
	IncrementCommentCount(context.Context, string) error
}
