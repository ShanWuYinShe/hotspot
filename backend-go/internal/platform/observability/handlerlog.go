package observability

import "net/http"

// LogServerError 记录 handler 层服务端错误（5xx/内部错误），
// 统一带 request_id 便于链路追踪。
func LogServerError(r *http.Request, method string, err error) {
	L().Error("handler error", "method", method, "error", err, "request_id", RequestIDFromContext(r.Context()))
}

// LogBusinessError 记录 handler 层业务错误（4xx 语义，客户端可读）。
func LogBusinessError(r *http.Request, method string, code int, message string) {
	// 业务错误统一经此出口：日志 + 指标（hotspot_business_errors_total）同步，
	// 避免 hand-written 错误响应漏观测（ops 审计 P0-3）。
	RecordBusinessError(r, code)
	L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", RequestIDFromContext(r.Context()))
}
