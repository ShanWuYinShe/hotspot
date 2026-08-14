package moments

import "errors"

var (
	ErrMomentNotFound   = errors.New("瞬间不存在")
	ErrPermissionDenied = errors.New("权限不足")
	ErrInvalidCursor    = errors.New("moments: invalid cursor")
	ErrTooManyImages    = errors.New("图片数量超过上限")
	ErrBlockedTag       = errors.New("moments: tags contain a blocked tag")
)
