package comments

import "errors"

var (
	ErrCommentNotFound  = errors.New("评论不存在")
	ErrPermissionDenied = errors.New("权限不足")
)
