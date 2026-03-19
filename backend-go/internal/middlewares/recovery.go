package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"backend-go/internal/api"
	apierr "backend-go/internal/errors"
)

// Recovery 捕获 Handler 中的 panic，记录日志并返回统一的 ApiResponse 错误封装。
func Recovery(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					fields := []zap.Field{
						zap.Any("panic", rec),
					}
					if reqID := middleware.GetReqID(r.Context()); reqID != "" {
						fields = append(fields, zap.String("request_id", reqID))
					}
					logger.Error("panic_recovered", fields...)

					// 使用统一的 ApiResponse 错误封装，符合 spec 中 HTTP 200 + code/message 规范。
					e := apierr.ErrInternal("")
					api.WriteError(w, r, e.Code, e.Message)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
