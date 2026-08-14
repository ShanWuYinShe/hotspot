package comments

import "errors"

var (
	ErrCommentNotFound    = errors.New("评论不存在")
	ErrPermissionDenied   = errors.New("权限不足")
	ErrMomentNoPermission = errors.New("comments: no permission to comment on this moment")
	// ErrCommentPrivacyDenied：瞬间作者限制评论（spec comment create → 10200 + privacy_setting）。
	ErrCommentPrivacyDenied = errors.New("comments: moment author restricts commenting")
	// ErrCommentBlockedByUser：被瞬间作者拉黑（spec comment create → 10200 + blocked_by_user）。
	ErrCommentBlockedByUser = errors.New("comments: caller is blocked by the moment author")
	// ErrInvalidCursor signals a malformed or unparseable pagination cursor.
	// Returned by repo cursor decoders so handlers can map it to the
	// generic "parameter error" code (httpx.CodeInvalidRequest) instead of the catch-all
	// httpx.CodeInternalUnavailable "service unavailable".
	ErrInvalidCursor = errors.New("comments: invalid cursor")
)
