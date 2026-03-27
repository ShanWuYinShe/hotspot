package auth

import (
	"net/http"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

var defaultService = NewService(config.Config{AuthTokenSecret: "dev-secret-change-me"})

func SetDefaultService(service *Service) {
	if service != nil {
		defaultService = service
	}
}

func DefaultService() *Service {
	return defaultService
}

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithPlatform(r.Context(), detectPlatform(r))
		if principal, expiresAt, ok := defaultService.AuthenticateRequest(ctx, r.Header.Get("Authorization"), readCookie(r, AccessTokenCookieName)); ok {
			ctx = WithPrincipal(ctx, principal)
			ctx = WithAccessExpiry(ctx, expiresAt)
			r = r.WithContext(ctx)
		} else {
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func RequiredAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithPlatform(r.Context(), detectPlatform(r))
		principal, expiresAt, ok := defaultService.AuthenticateRequest(ctx, r.Header.Get("Authorization"), readCookie(r, AccessTokenCookieName))
		if !ok {
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
