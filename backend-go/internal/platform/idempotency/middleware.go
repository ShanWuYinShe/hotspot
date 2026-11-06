package idempotency

import (
	"net/http"
	"regexp"

	"hotspot/backend-go/internal/platform/httpx"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,64}$`)

// Middleware validates the Idempotency-Key header and injects it into the request context.
// All store checking and response replay is handled by ReplayOrStore inside the handler.
// This avoids duplicating store logic with a mismatched scope key.
//
// 契约（api_spec_base.md「服务端行为」）：格式非法的非空 key 返回 22000（客户端
// 应更换新 key 重试）——静默放行会让客户端误以为请求受幂等保护。缺失 key 合法
// （幂等键仅 MUST 端点要求，由 ReplayOrStore 校验）。
// WithStore 是根中间件：把装配期构造的 Store 注入请求 context，供
// Middleware/ReplayOrStore 读取。它不做任何幂等判定，可挂在路由树最外层。
func WithStore(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if store == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithStoreContext(r.Context(), store)))
		})
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !keyPattern.MatchString(key) {
			httpx.WriteCodeError(w, r, httpx.CodeIdempotencyConflict, "Idempotency-Key 格式非法，请更换新 key 重试")
			return
		}
		r = r.WithContext(WithKey(r.Context(), key))
		next.ServeHTTP(w, r)
	})
}
