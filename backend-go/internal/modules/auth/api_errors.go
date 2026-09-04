package auth

import (
	"errors"

	"hotspot/backend-go/internal/platform/httpx"
)

var (
	ErrUnauthenticated                = errors.New("auth: unauthenticated")
	ErrLoginIDConflict                = errors.New("auth: login id conflict")
	ErrInvalidCredentials             = errors.New("auth: invalid credentials")
	ErrInvalidOldPassword             = errors.New("auth: invalid old password")
	ErrPasswordResetNotFound          = errors.New("auth: password reset not found")
	ErrPasswordResetThrottled         = errors.New("auth: password reset throttled")
	ErrVerificationServiceUnavailable = errors.New("auth: verification service not available")
	ErrAccountLocked                  = errors.New("auth: account locked due to too many failed attempts")
	// ErrAccountBanned 封禁账号尝试登录（spec api_spec_errors.md → 10201：
	// 客户端须强制登出并展示专用封禁页，与普通错误 toast 语义不同）。
	ErrAccountBanned = errors.New("auth: account banned")
	// ErrInvalidRegisterCode 注册验证码错误/已过期（spec api_spec_public.md →
	// register：10401 旧凭证校验失败）。
	ErrInvalidRegisterCode = errors.New("auth: invalid register verification code")
	// ErrMissingRegisterCode 手机号/邮箱型注册缺验证码（参数缺失 → 10000，
	// 与「验证码错误」的 10401 区分）。
	ErrMissingRegisterCode = errors.New("auth: missing register verification code")
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
	case errors.Is(err, ErrUnauthenticated):
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "Access Token 无效或过期"}
	case errors.Is(err, ErrLoginIDConflict):
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "账号已存在"}
	case errors.Is(err, ErrInvalidCredentials):
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "账号或密码错误"}
	case errors.Is(err, ErrAccountLocked):
		// 10500 须携带 Retry-After（spec api_spec_errors.md → 10500）；
		// 锁定窗口为服务端固定值，取完整窗口时长作为保守重试等待。
		return &httpx.APIError{Code: httpx.CodeRateLimited, Message: "登录失败次数过多，请15分钟后重试", RetryAfterSeconds: int(defaultLockDuration.Seconds())}
	case errors.Is(err, ErrAccountBanned):
		// 10201（spec api_spec_errors.md）：账号被封禁，客户端强制登出 + 封禁页
		return &httpx.APIError{Code: httpx.CodeAccountBanned, Message: "账号已被封禁"}
	case errors.Is(err, ErrInvalidOldPassword):
		return &httpx.APIError{Code: httpx.CodeInvalidOldCredential, Message: "旧密码错误"}
	case errors.Is(err, ErrPasswordResetNotFound):
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "重置请求不存在或已过期"}
	case errors.Is(err, ErrPasswordResetThrottled):
		// 同上：窗口自最近一次请求起算，取完整窗口作为保守等待。
		return &httpx.APIError{Code: httpx.CodeRateLimited, Message: "请求过于频繁", RetryAfterSeconds: int(passwordResetThrottleWindow.Seconds())}
	case errors.Is(err, ErrVerificationServiceUnavailable):
		// 验证码服务不可用属服务端错误（spec 10600），非客户端限流（10500）
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "验证码服务暂不可用"}
	case errors.Is(err, ErrInvalidRegisterCode):
		// 验证码错误/过期：10401 旧凭证校验失败（spec api_spec_public.md → register）
		return &httpx.APIError{Code: httpx.CodeInvalidOldCredential, Message: "验证码错误或已过期"}
	case errors.Is(err, ErrMissingRegisterCode):
		// 手机号/邮箱型注册缺验证码属参数缺失（10000）
		return &httpx.APIError{Code: httpx.CodeInvalidRequest, Message: "参数错误"}
	default:
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
	}
}
