package auth

import (
	"net/http"
	"sync/atomic"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

// defaultService is written once at boot and read on every request, so guard
// it with an atomic pointer instead of a bare package global (which races).
var defaultService atomic.Pointer[Service]

func init() {
	defaultService.Store(NewService(config.Config{AuthTokenSecret: config.DevDefaultAuthTokenSecret()}))
}

func SetDefaultService(service *Service) {
	if service != nil {
		defaultService.Store(service)
	}
}

func DefaultService() *Service {
	return defaultService.Load()
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithPlatform(r.Context(), DetectPlatform(r, ""))
		if principal, expiresAt, ok := defaultService.Load().AuthenticateRequest(ctx, r.Header.Get("Authorization"), ReadCookie(r, AccessTokenCookieName)); ok {
			ctx = WithPrincipal(ctx, principal)
			ctx = WithAccessExpiry(ctx, expiresAt)
			r = r.WithContext(ctx)
		} else {
			observability.L().Debug("optional auth: no valid token, downgrade to anonymous",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
			)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func RequiredAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithPlatform(r.Context(), DetectPlatform(r, ""))
		svc := defaultService.Load()
		principal, expiresAt, ok := svc.AuthenticateRequest(ctx, r.Header.Get("Authorization"), ReadCookie(r, AccessTokenCookieName))
		if !ok {
			hasHeader := r.Header.Get("Authorization") != ""
			hasCookie := ReadCookie(r, AccessTokenCookieName) != ""
			observability.L().Info("required auth failed: invalid or expired token",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"has_auth_header", hasHeader,
				"has_cookie", hasCookie,
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeUnauthorized, "Access Token 无效或过期"))
			return
		}
		// Banned accounts are force-logged-out and get httpx.CodeAccountBanned so the client can
		// show the dedicated ban page (spec/api_spec_errors.md). Web clients
		// also get their auth cookies cleared.
		if svc.IsUserBanned(ctx, principal.UserID) {
			ClearAuthCookies(w)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeAccountBanned, "账号已被封禁"))
			return
		}
		ctx = WithPrincipal(ctx, principal)
		ctx = WithAccessExpiry(ctx, expiresAt)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// DetectPlatform 判定请求端平台（web/app/anonymous），单一实现供 HTTP 鉴权
// 中间件与 realtime WS 鉴权复用，防两处语义漂移。clientTypeHint 非空时信任
// 显式声明（WS Auth payload 自报端型），否则按 access cookie / Authorization
// 头推断——cookie 优先于 bearer：双端并存登录时 web cookie 是用户主动选择的
// 本站会话。
func DetectPlatform(r *http.Request, clientTypeHint string) string {
	if clientTypeHint != "" {
		return clientTypeHint
	}
	if ReadCookie(r, AccessTokenCookieName) != "" {
		return "web"
	}
	if r.Header.Get("Authorization") != "" {
		return "app"
	}
	return "anonymous"
}

// ReadCookie 读取请求 cookie，缺失时返回空串（调用方只做存在性/值判断）。
func ReadCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
