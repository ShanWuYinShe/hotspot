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
		ctx := WithPlatform(r.Context(), detectPlatform(r))
		if principal, expiresAt, ok := defaultService.Load().AuthenticateRequest(ctx, r.Header.Get("Authorization"), readCookie(r, AccessTokenCookieName)); ok {
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
		ctx := WithPlatform(r.Context(), detectPlatform(r))
		principal, expiresAt, ok := defaultService.Load().AuthenticateRequest(ctx, r.Header.Get("Authorization"), readCookie(r, AccessTokenCookieName))
		if !ok {
			hasHeader := r.Header.Get("Authorization") != ""
			hasCookie := readCookie(r, AccessTokenCookieName) != ""
			observability.L().Info("required auth failed: invalid or expired token",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"has_auth_header", hasHeader,
				"has_cookie", hasCookie,
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10100, "Access Token 无效或过期"))
			return
		}
		ctx = WithPrincipal(ctx, principal)
		ctx = WithAccessExpiry(ctx, expiresAt)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminAuth(next http.Handler) http.Handler {
	return AdminAccess(next)
}

func AdminAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || !principal.IsAdmin() {
			userID := ""
			if ok {
				userID = principal.UserID
			}
			observability.L().Warn("admin access denied: not an admin",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"user_id", userID,
				"has_principal", ok,
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "非系统管理员"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func detectPlatform(r *http.Request) string {
	if readCookie(r, AccessTokenCookieName) != "" {
		return "web"
	}
	if r.Header.Get("Authorization") != "" {
		return "app"
	}
	return "anonymous"
}

func readCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
