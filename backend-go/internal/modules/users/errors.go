package users

import "errors"

var (
	ErrUnauthorized     = errors.New("users: unauthorized")
	ErrUserNotFound     = errors.New("users: user not found")
	ErrPermissionDenied = errors.New("权限不足")
	ErrConflict         = errors.New("users: conflict")
	ErrInvalidCursor    = errors.New("users: invalid cursor")
	// ErrInvalidRequest 参数错误（10000）。用于缺目标 ID 等必传参数缺失。
	ErrInvalidRequest = errors.New("users: invalid request")
	// ErrBlockedTag 用户资料 tags 含被屏蔽标签（10400）。
	ErrBlockedTag = errors.New("users: tags contain blocked tag")
)
