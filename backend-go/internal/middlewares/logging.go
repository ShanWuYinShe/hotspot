package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Logging 使用 zap 记录每个 HTTP 请求，并附带 request_id 等关键字段。
func Logging(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			next.ServeHTTP(ww, r)

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
			}
			if reqID := middleware.GetReqID(r.Context()); reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}

			logger.Info("http_request", fields...)
		})
	}
}
