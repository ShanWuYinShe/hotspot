package httpx

import (
	"net/http"
	"runtime/debug"

	"hotspot/backend-go/internal/platform/observability"
)

// MaxRequestBodyBytes caps JSON request bodies to protect against memory
// exhaustion from oversized payloads. 1 MiB is generous for these RPC-style
// endpoints (the largest legitimate body is a multi-field create request).
const MaxRequestBodyBytes = 1 << 20

// LimitRequestBody wraps r.Body in a MaxBytesReader so handlers that decode
// JSON cannot be fed an unbounded payload. Mount as near-router middleware.
func LimitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// Recoverer catches handler panics and returns a JSON envelope (code CodeInternalUnavailable)
// instead of letting chi/net-http emit a plaintext 500 with a Go stack trace.
// The spec (api_spec_base.md) forbids exposing stack traces to clients.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				observability.L().Error("http panic recovered",
					"panic", rec,
					"request_id", observability.RequestIDFromContext(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				WriteJSON(w, http.StatusOK, Error(CodeInternalUnavailable, "服务暂时不可用"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// NotFoundHandler returns the standard envelope for unknown routes. The spec
// mandates all business responses be HTTP 200 + ApiResponse; a bare 404 would
// leak that the path-matching layer is chi and confuse envelope-only clients.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, Error(CodeNotFound, "资源不存在"))
}

// MethodNotAllowedHandler returns the envelope for wrong-method requests.
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, Error(CodeInvalidRequest, "请求方法不被支持"))
}
