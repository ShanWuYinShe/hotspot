package observability

import (
	"net/http"
	"time"
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
				"client_ip", clientIP(r),
				"bytes", rc.bytes,
			}

			switch {
			case rc.status >= 500:
				logger.Error("request completed (server error)", fields...)
			case rc.status >= 400:
				logger.Warn("request completed (client error)", fields...)
			default:
				logger.Info("request completed", fields...)
			}
		})
	}
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
