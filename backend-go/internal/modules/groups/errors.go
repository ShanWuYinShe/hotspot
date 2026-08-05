package groups

import "errors"

var (
	ErrAccessRestricted     = errors.New("权限不足")
	ErrPermissionDenied     = ErrAccessRestricted
	ErrGroupNotFound        = errors.New("群组不存在")
	ErrUserNotFound         = errors.New("用户不存在")
	ErrBusinessConflict     = errors.New("业务冲突")
	ErrAlreadyMember        = errors.New("已是群成员")
	ErrAlreadyBanned        = errors.New("已被封禁")
	ErrNotMember            = errors.New("不是群成员")
	ErrAlreadyApplied       = errors.New("已申请过")
	ErrAlreadyProcessed     = errors.New("已处理")
	ErrInvalidPassword      = errors.New("密码错误")
	ErrGroupFull            = errors.New("群组已满")
	ErrInviteExpired        = errors.New("邀请已过期")
	ErrTargetNotMember      = errors.New("目标用户不是群成员")
	ErrApplicationNotFound  = errors.New("申请不存在")
	ErrAnnouncementNotFound = errors.New("公告不存在")
	ErrInvalidCursor        = errors.New("groups: invalid cursor")
)
