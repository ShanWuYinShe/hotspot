package auth

import (
	"context"
	"time"
)

// ctxKey 用于在 context 中存储认证相关信息，保持包内私有以避免冲突。
type ctxKey string

const authInfoKey ctxKey = "auth_info"

// AuthInfo 表示当前请求的认证上下文，由认证中间件解析 token 后注入。
//
// - UserID/SessionID：来自 access_token/refresh_token 的 claims
// - AccessToken/RefreshToken：原始 token 字符串（用于写入黑名单）
// - AccessExpiresAt/RefreshExpiresAt：对应 token 的过期时间
type AuthInfo struct {
	UserID           string
	SessionID        string
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// WithAuthInfo 将认证信息写入 context，供 Service 层和 Handler 读取。
func WithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	if info == nil {
		return ctx
	}
	return context.WithValue(ctx, authInfoKey, info)
}

// FromContext 从 context 中提取认证信息。
func FromContext(ctx context.Context) (*AuthInfo, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(authInfoKey); v != nil {
		if info, ok := v.(*AuthInfo); ok {
			return info, true
		}
	}
	return nil, false
}
