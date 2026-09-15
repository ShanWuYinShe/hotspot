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
	// ErrCommentTooLong：评论内容超过 ClientConfig comment_max_length（spec 10400）。
	ErrCommentTooLong = errors.New("comments: content exceeds max length")
	// ErrTooManyMentions：mentions 数量超过 api_spec_base.md 字段校验约束上限 20（spec 10000）。
	// 与 moments/groupmessages 对齐：不限制会放大通知写入（每个 mention 触发一次
	// 通知落库 + 一次 WS 推送），构成通知轰炸向量。
	ErrTooManyMentions = errors.New("comments: mentions exceed max count")
	// ErrInvalidCursor signals a malformed or unparseable pagination cursor.
	// Returned by repo cursor decoders so handlers can map it to the
	// generic "parameter error" code (httpx.CodeInvalidRequest) instead of the catch-all
	// httpx.CodeInternalUnavailable "service unavailable".
	ErrInvalidCursor = errors.New("comments: invalid cursor")
)
