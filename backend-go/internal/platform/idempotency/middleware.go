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
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !keyPattern.MatchString(key) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeIdempotencyConflict, "Idempotency-Key 格式非法，请更换新 key 重试"))
			return
		}
		r = r.WithContext(WithKey(r.Context(), key))
		next.ServeHTTP(w, r)
	})
}
