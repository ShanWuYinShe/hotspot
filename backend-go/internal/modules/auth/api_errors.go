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
	ErrVerificationServiceUnavailable = errors.New("auth: verification service not available")
	ErrAccountLocked           = errors.New("auth: account locked due to too many failed attempts")
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
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "账号已存在"}
	case errors.Is(err, ErrInvalidCredentials):
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "账号或密码错误"}
	case errors.Is(err, ErrAccountLocked):
		return &httpx.APIError{Code: httpx.CodeRateLimited, Message: "登录失败次数过多，请15分钟后重试"}
	case errors.Is(err, ErrInvalidOldPassword):
		return &httpx.APIError{Code: httpx.CodeInvalidOldCredential, Message: "旧密码错误"}
	case errors.Is(err, ErrPasswordResetNotFound):
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "重置请求不存在或已过期"}
	case errors.Is(err, ErrPasswordResetThrottled):
		return &httpx.APIError{Code: httpx.CodeRateLimited, Message: "请求过于频繁"}
	case errors.Is(err, ErrVerificationServiceUnavailable):
		return &httpx.APIError{Code: httpx.CodeRateLimited, Message: "验证码服务暂不可用"}
	default:
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
	}
}
