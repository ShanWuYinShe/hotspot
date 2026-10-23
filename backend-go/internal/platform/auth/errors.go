package auth

import (
	"errors"

	"hotspot/backend-go/internal/platform/httpx"
)

// ErrDeviceSessionConflict 表示「同一 (user_id, device_id) 已有活跃会话」——
// 由 auth_sessions 的部分唯一索引 uq_auth_sessions_active_device 触发
// （迁移 057）。调用方（Login/Guest）据此撤销旧会话后重试，从根上消除
// 「先撤销再新建」两步之间的 TOCTOU 竞态。
var ErrDeviceSessionConflict = errors.New("auth: active session already exists for device")

func Unauthorized(message string) *httpx.APIError {
	return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: message}
}

func Forbidden(message string) *httpx.APIError {
	return httpx.Forbidden(httpx.ReasonNotResourceOwner, message)
}
