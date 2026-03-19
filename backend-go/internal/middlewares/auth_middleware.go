package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"backend-go/internal/api"
	"backend-go/internal/auth"
	apierr "backend-go/internal/errors"
	"backend-go/internal/repository"
)

// extractAccessToken 从 Authorization Header 或 Cookie 中提取 access_token。
//
// 优先使用 `Authorization: Bearer <token>`（App 端）；
// 若不存在，则回退到 `access_token` Cookie（Web 端）。
func extractAccessToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if c, err := r.Cookie("access_token"); err == nil {
		return c.Value
	}
	return ""
}

// attachAuthInfo 根据 token 解析并在 context 中注入 AuthInfo。返回新的请求和是否成功注入。
func attachAuthInfo(r *http.Request, tokens auth.TokenManager, blacklists repository.TokenBlacklistRepository) (*http.Request, bool) {
	tokenStr := extractAccessToken(r)
	if tokenStr == "" {
		return r, false
	}

	// 黑名单检查（若 Redis 出错，则视为 token 不可用）。
	if blacklists != nil {
		if blacklisted, err := blacklists.IsTokenBlacklisted(r.Context(), tokenStr); err != nil || blacklisted {
			return r, false
		}
	}

	userID, sessionID, exp, err := tokens.ParseAccessToken(tokenStr)
	if err != nil {
		return r, false
	}

	info := &auth.AuthInfo{
		UserID:           userID,
		SessionID:        sessionID,
		AccessToken:      tokenStr,
		AccessExpiresAt:  exp,
		RefreshToken:     "",          // 仅由特定 Handler 在需要时补充
		RefreshExpiresAt: time.Time{}, // 由调用者按需填充
	}
	ctx := auth.WithAuthInfo(r.Context(), info)
	return r.WithContext(ctx), true
}

// AuthOptional 为 Token 可选场景提供认证中间件：
// - 若不存在/无效/已拉黑的 Token，则不注入 AuthInfo，直接放行；
// - 若 Token 合法，则在 context 中注入 AuthInfo。
func AuthOptional(tokens auth.TokenManager, blacklists repository.TokenBlacklistRepository) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 为保证 trace 完整度，优先确保 RequestID 已存在。
			if middleware.GetReqID(r.Context()) == "" {
				// 若上游未注入，则不在此处强行生成，只执行业务逻辑。
			}

			if tokens == nil {
				next.ServeHTTP(w, r)
				return
			}

			if reqWithAuth, ok := attachAuthInfo(r, tokens, blacklists); ok {
				next.ServeHTTP(w, reqWithAuth)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthRequired 为 Token 必须场景提供认证中间件：
// - 若 Token 缺失/无效/已拉黑，则返回 10100（auth_failed），不进入后续 Handler；
// - 若 Token 合法，则在 context 中注入 AuthInfo 并继续处理。
func AuthRequired(tokens auth.TokenManager, blacklists repository.TokenBlacklistRepository) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokens == nil {
				e := apierr.New(10100, "auth_failed")
				api.WriteError(w, r, e.Code, e.Message)
				return
			}

			reqWithAuth, ok := attachAuthInfo(r, tokens, blacklists)
			if !ok {
				e := apierr.New(10100, "auth_failed")
				api.WriteError(w, r, e.Code, e.Message)
				return
			}
			next.ServeHTTP(w, reqWithAuth)
		})
	}
}
