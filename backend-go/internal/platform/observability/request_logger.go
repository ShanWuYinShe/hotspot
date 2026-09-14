package observability

import (
	"net/http"
	"time"

	"hotspot/backend-go/internal/platform/requestmeta"
)

type responseCapture struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	n, err := rc.ResponseWriter.Write(b)
	rc.bytes += n
	return n, err
}

// slowRequestLogThreshold 及以上延迟的成功请求保留 Info 级访问日志。
// 取舍（审计 perf P3-3）：成功且快的请求全量记 Info 会在高 QPS 下淹没真正
// 需要关注的信号（错误/慢请求）并放大日志量，故降为 Debug——线上 log level
// 为 info 时不输出，本地/排查时可调低 level 全量查看；错误（>=400）与慢请求
// 保留原有级别，可观测性不受影响。
const slowRequestLogThreshold = 500 * time.Millisecond

func RequestLoggerMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}
			requestID := RequestIDFromContext(r.Context())

			next.ServeHTTP(rc, r)

			elapsed := time.Since(start)
			fields := []any{
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"status", rc.status,
				"latency_ms", elapsed.Milliseconds(),
				"request_id", requestID,
				"client_ip", requestmeta.ClientIP(r),
				"bytes", rc.bytes,
			}
			if cfRay := requestmeta.RequestRay(r); cfRay != "" {
				fields = append(fields, "cf_ray", cfRay)
			}
			if country := requestmeta.ClientCountry(r); country != "" {
				fields = append(fields, "cf_country", country)
			}
			if proto := requestmeta.ForwardedProto(r); proto != "" {
				fields = append(fields, "forwarded_proto", proto)
			}

			switch {
			case rc.status >= 500:
				logger.Error("request completed (server error)", fields...)
			case rc.status >= 400:
				logger.Warn("request completed (client error)", fields...)
			case elapsed >= slowRequestLogThreshold:
				logger.Info("request completed (slow)", fields...)
			default:
				logger.Debug("request completed", fields...)
			}
		})
	}
}
