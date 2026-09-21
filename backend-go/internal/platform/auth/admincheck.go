package auth

import (
	"context"
	"net/http"
	"sync/atomic"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

// AdminChecker 判定 userID 是否为系统管理员（真实验证，查询 admin_users 表）。
// 由 bootstrap 注入 admin 模块实现；未注入时 IsAdminUser 返回 false（fail-closed，
// 管理端点是安全边界，宁拒绝勿放行）。
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

var adminChecker atomic.Value // *AdminChecker

// SetAdminChecker 注入管理员校验实现（组合根调用，通常在模块装配阶段）。
func SetAdminChecker(checker AdminChecker) {
	adminChecker.Store(&checker)
}

// IsAdminUser 返回 userID 是否为管理员。checker 未配置或查询出错均视为非管理员
// （fail-closed）。
func IsAdminUser(ctx context.Context, userID string) bool {
	value := adminChecker.Load()
	if value == nil {
		return false
	}
	checkerPtr, ok := value.(*AdminChecker)
	if !ok || checkerPtr == nil {
		return false
	}
	checker := *checkerPtr
	isAdmin, err := checker.IsAdmin(ctx, userID)
	return err == nil && isAdmin
}

// AdminAccess 校验当前 principal 是否为平台管理员（httpx.CodeForbidden/非平台管理员）。
// 管理端点统一挂该中间件（admin 模块 /admin/api/v1 分组）。
func AdminAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			observability.L().Warn("admin access denied: no principal",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(httpx.CodeForbidden, "非平台管理员", httpx.Forbidden(httpx.ReasonNotAdmin, "非平台管理员").Data))
			return
		}
		if !IsAdminUser(r.Context(), principal.UserID) {
			observability.L().Warn("admin access denied: not an admin",
				"method", r.Method,
				"uri", r.URL.RequestURI(),
				"user_id", principal.UserID,
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(httpx.CodeForbidden, "非平台管理员", httpx.Forbidden(httpx.ReasonNotAdmin, "非平台管理员").Data))
			return
		}
		next.ServeHTTP(w, r)
	})
}
