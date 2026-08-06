package comments

import "errors"

var (
	ErrCommentNotFound    = errors.New("评论不存在")
	ErrPermissionDenied   = errors.New("权限不足")
	ErrMomentNoPermission = errors.New("comments: no permission to comment on this moment")
	// ErrInvalidCursor signals a malformed or unparseable pagination cursor.
	// Returned by repo cursor decoders so handlers can map it to the
	// generic "parameter error" code (httpx.CodeInvalidRequest) instead of the catch-all
	// httpx.CodeInternalUnavailable "service unavailable".
	ErrInvalidCursor = errors.New("comments: invalid cursor")
)
