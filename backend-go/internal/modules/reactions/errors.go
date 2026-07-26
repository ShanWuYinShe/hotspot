package reactions

import "errors"

var (
	// ErrReactionTargetNotFound maps to HTTP code 10300 (resource not found).
	ErrReactionTargetNotFound = errors.New("反应目标不存在")
	// ErrInvalidEmoji signals an emoji outside the server whitelist; maps to 10000.
	ErrInvalidEmoji = errors.New("reactions: invalid emoji")
	// ErrInvalidCursor signals a malformed pagination cursor; maps to 10000.
	ErrInvalidCursor = errors.New("reactions: invalid cursor")
)
