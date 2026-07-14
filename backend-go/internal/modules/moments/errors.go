package moments

import "errors"

var (
	ErrMomentNotFound   = errors.New("瞬间不存在")
	ErrPermissionDenied = errors.New("权限不足")
	ErrInvalidCursor    = errors.New("无效的分页游标")
	ErrTooManyImages    = errors.New("图片数量超过上限")
)
