package auth

import (
	"context"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

// requestDeps 是根中间件注入请求 context 的平台认证依赖：Service（验签/
// 会话）与 AdminChecker（管理端判定）。不再有进程级 DefaultService 单例，
// 同进程可装配多套 App。
type requestDeps struct {
	service      *Service
	adminChecker AdminChecker
}

type depsContextKey struct{}

// WithDependencies 是根中间件：把装配期构造的认证依赖注入请求 context。
func WithDependencies(service *Service, adminChecker AdminChecker) func(http.Handler) http.Handler {
	deps := requestDeps{service: service, adminChecker: adminChecker}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), depsContextKey{}, deps)))
		})
	}
}

func depsFromContext(ctx context.Context) requestDeps {
	deps, _ := ctx.Value(depsContextKey{}).(requestDeps)
	return deps
}

func serviceFromContext(ctx context.Context) *Service { return depsFromContext(ctx).service }

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithPlatform(r.Context(), DetectPlatform(r, ""))
		svc := serviceFromContext(r.Context())
		if svc == nil {
			// 未装配认证依赖时匿名降级（生产由根中间件 WithDependencies 注入）。
			observability.L().Error("optional auth: auth service not wired")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if principal, expiresAt, ok := svc.AuthenticateRequest(ctx, r.Header.Get("Authorization"), ReadCookie(r, AccessTokenCookieName)); ok {
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
		svc := serviceFromContext(r.Context())
		if svc == nil {
			// 安全边界 fail-closed：生产装配漏注入时拒绝而不是放行。
			observability.L().Error("required auth: auth service not wired", "method", r.Method, "uri", r.URL.RequestURI())
			httpx.WriteCodeError(w, r, httpx.CodeInternalUnavailable, "服务暂时不可用")
			return
		}
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
			httpx.WriteCodeError(w, r, httpx.CodeUnauthorized, "Access Token 无效或过期")
			return
		}
		// Banned accounts are force-logged-out and get httpx.CodeAccountBanned so the client can
		// show the dedicated ban page (spec/api_spec_errors.md). Web clients
		// also get their auth cookies cleared.
		if svc.IsUserBanned(ctx, principal.UserID) {
			ClearAuthCookies(w)
			httpx.WriteCodeError(w, r, httpx.CodeAccountBanned, "账号已被封禁")
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
