package moments

import "errors"

var (
	ErrMomentNotFound   = errors.New("瞬间不存在")
	ErrPermissionDenied = errors.New("权限不足")
	ErrInvalidCursor    = errors.New("moments: invalid cursor")
	ErrTooManyImages    = errors.New("图片数量超过上限")
	// ErrDescriptionTooLong：动态正文超过 ClientConfig moment_description_max_length（spec 10400）。
	ErrDescriptionTooLong = errors.New("moments: description exceeds max length")
	// ErrTooManyMentions：mentions 数量超过 api_spec_base.md 字段校验约束上限 20（spec 10000）。
	ErrTooManyMentions = errors.New("moments: mentions exceed max count")
	// ErrTooManyTags：tags 数量超过 api_spec_base.md 字段校验约束上限 10（spec 10000）。
	// 不限制会让单条请求放大为数千次 tag 逐条 INSERT。
	ErrTooManyTags = errors.New("moments: tags exceed max count")
	ErrBlockedTag  = errors.New("moments: tags contain a blocked tag")
	// ErrInvalidFetchRequest 发现流（fetch_following/fetch_nearby）请求非法：
	// 未登录访问 fetch_following、reference_point 坐标非法等。
	ErrInvalidFetchRequest = errors.New("moments: invalid fetch request")
)
