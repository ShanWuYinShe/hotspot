package moments

import "errors"

var (
	ErrMomentNotFound   = errors.New("瞬间不存在")
	ErrPermissionDenied = errors.New("权限不足")
	ErrInvalidCursor    = errors.New("moments: invalid cursor")
	ErrTooManyImages    = errors.New("图片数量超过上限")
	ErrBlockedTag       = errors.New("moments: tags contain a blocked tag")
	// ErrInvalidFetchRequest 发现流（fetch_following/fetch_nearby）请求非法：
	// 未登录访问 fetch_following、reference_point 坐标非法等。
	ErrInvalidFetchRequest = errors.New("moments: invalid fetch request")
)
