package users

import "errors"

var (
	ErrUnauthorized             = errors.New("users: unauthorized")
	ErrUserNotFound             = errors.New("users: user not found")
	ErrPermissionDenied         = errors.New("users: permission denied")
	ErrConflict                 = errors.New("users: conflict")
	ErrInvalidCursor            = errors.New("users: invalid cursor")
	ErrContactChangeUnavailable = errors.New("users: contact change unavailable")
)
