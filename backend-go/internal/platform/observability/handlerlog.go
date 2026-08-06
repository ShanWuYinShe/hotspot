package observability

import "net/http"

// LogServerError 记录 handler 层服务端错误（5xx/内部错误），
// 统一带 request_id 便于链路追踪。
func LogServerError(r *http.Request, method string, err error) {
	L().Error("handler error", "method", method, "error", err, "request_id", RequestIDFromContext(r.Context()))
}

// LogBusinessError 记录 handler 层业务错误（4xx 语义，客户端可读）。
func LogBusinessError(r *http.Request, method string, code int, message string) {
	L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", RequestIDFromContext(r.Context()))
}
