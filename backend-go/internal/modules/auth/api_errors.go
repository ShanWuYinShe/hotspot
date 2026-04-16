package auth

import (
	"errors"

	"hotspot/backend-go/internal/platform/httpx"
)

var (
	ErrLoginIDConflict         = errors.New("auth: login id conflict")
	ErrInvalidCredentials      = errors.New("auth: invalid credentials")
	ErrInvalidOldPassword      = errors.New("auth: invalid old password")
	ErrPasswordResetNotFound   = errors.New("auth: password reset not found")
	ErrPasswordResetThrottled  = errors.New("auth: password reset throttled")
	ErrEmailServiceUnavailable = errors.New("auth: email service not available")
)

func apiErrorFrom(err error) *httpx.APIError {
	if err == nil {
		return nil
	}
	var apiErr *httpx.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	switch {
	case errors.Is(err, ErrLoginIDConflict):
		return &httpx.APIError{Code: 10400, Message: "账号已存在"}
	case errors.Is(err, ErrInvalidCredentials):
		return &httpx.APIError{Code: 10100, Message: "账号或密码错误"}
	case errors.Is(err, ErrInvalidOldPassword):
		return &httpx.APIError{Code: 10100, Message: "旧密码错误"}
	case errors.Is(err, ErrPasswordResetNotFound):
		return &httpx.APIError{Code: 10300, Message: "重置请求不存在或已过期"}
	case errors.Is(err, ErrPasswordResetThrottled):
		return &httpx.APIError{Code: 10500, Message: "请求过于频繁"}
	case errors.Is(err, ErrEmailServiceUnavailable):
		return &httpx.APIError{Code: 10500, Message: "邮件服务暂不可用"}
	default:
		return &httpx.APIError{Code: 10600, Message: "服务暂时不可用"}
	}
}
