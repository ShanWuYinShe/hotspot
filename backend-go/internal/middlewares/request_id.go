package middlewares

import (
	"net/http"

	chimid "github.com/go-chi/chi/v5/middleware"
)

// RequestID 为每个请求注入唯一的请求 ID，
// 使用 chi 的 RequestID 实现，并确保通过 X-Request-ID 头暴露给下游。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 先通过 chi 的 RequestID 中间件生成/提取请求 ID。
		chimid.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reqID := chimid.GetReqID(r.Context()); reqID != "" {
				w.Header().Set("X-Request-ID", reqID)
			}
			// 继续后续链路。
			next.ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})
}
