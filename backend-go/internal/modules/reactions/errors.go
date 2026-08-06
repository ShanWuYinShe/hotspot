package reactions

import "errors"

var (
	// ErrReactionTargetNotFound maps to HTTP code httpx.CodeNotFound (resource not found).
	ErrReactionTargetNotFound = errors.New("反应目标不存在")
	// ErrInvalidEmoji signals an emoji outside the server whitelist; maps to httpx.CodeInvalidRequest.
	ErrInvalidEmoji = errors.New("reactions: invalid emoji")
	// ErrInvalidCursor signals a malformed pagination cursor; maps to httpx.CodeInvalidRequest.
	ErrInvalidCursor = errors.New("reactions: invalid cursor")
)
