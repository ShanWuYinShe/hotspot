package groups

import "errors"

var (
	ErrGroupNotFound    = errors.New("群组不存在")
	ErrPermissionDenied = errors.New("权限不足")
	ErrUserNotFound     = errors.New("用户不存在")
	ErrAlreadyMember    = errors.New("已是群成员")
	ErrAlreadyBanned    = errors.New("已被封禁")
	ErrInvalidPassword  = errors.New("密码错误")
	ErrTargetNotMember  = errors.New("目标用户不是群成员")
)
