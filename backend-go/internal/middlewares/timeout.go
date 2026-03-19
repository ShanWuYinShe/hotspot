package middlewares

import (
	"net/http"
	"time"

	chimid "github.com/go-chi/chi/v5/middleware"
)

// Timeout 为请求上下文设置超时时间，超时行为由 chi 的 Timeout 中间件负责。
// 业务 Handler 应尊重 r.Context().Done()，必要时可在更高层将 503 映射到统一错误码。
func Timeout(d time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return chimid.Timeout(d)(next)
	}
}
