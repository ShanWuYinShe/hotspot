package users

import "errors"

var (
	ErrUnauthorized     = errors.New("users: unauthorized")
	ErrUserNotFound     = errors.New("users: user not found")
	ErrPermissionDenied = errors.New("权限不足")
	ErrConflict         = errors.New("users: conflict")
	ErrInvalidCursor    = errors.New("users: invalid cursor")
)
