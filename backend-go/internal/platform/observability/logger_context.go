package observability

import (
	"context"
	"net/http"
)

// loggerContextKey 是请求级 Logger 注入的 context key。
type loggerContextKey struct{}

// WithLogger 把应用级 Logger 注入 context。组合根中间件在请求入口注入一次，
// 模块内日志统一 FromContext(ctx) 获取——不再跨包直取进程级 defaultLogger，
// 嵌入/多 App 场景可各自持有独立日志实例。
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// FromContext 返回请求级 Logger；未注入（后台 goroutine/直调测试）时回退
// 进程级 L()，保持既有日志行为不变。
func FromContext(ctx context.Context) *Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerContextKey{}).(*Logger); ok && logger != nil {
			return logger
		}
	}
	return L()
}

// WithLoggerMiddleware 把 Logger 注入每个请求的 context。
func WithLoggerMiddleware(logger *Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = L()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithLogger(r.Context(), logger)))
		})
	}
}
