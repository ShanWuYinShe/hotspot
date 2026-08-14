package groups

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/rbac"
)

// groupAdminGuard 是 /api/v1/groups/admin 守卫子树的统一守卫中间件（spec api_spec_authz.md）：
// 调用者必须是请求目标群的 owner/admin；owner 专属端点（dismiss、transfer_ownership）
// 再判一次 owner。非成员 → 10200+not_group_member；成员非管理 → 10200+not_group_admin。
//
// 角色解析经包级注入的 serviceGroupRoleResolver（bootstrap 接线 Service.RoleOf）；
// 未注入时 fail-closed（10600）。
func groupAdminGuard(ownerOnly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groupID, ok := guardGroupID(r)
			if !ok || groupID == "" {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
				return
			}
			if serviceGroupRoleResolver == nil {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
				return
			}
			role, found, err := serviceGroupRoleResolver.RoleOf(r.Context(), groupID)
			if err != nil {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
				return
			}
			if !found {
				httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(httpx.CodeForbidden, "非群成员", httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonNotGroupMember}))
				return
			}
			if !rbac.GroupRole(role).IsAdminRole() {
				httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(httpx.CodeForbidden, "非群管理员", httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonNotGroupAdmin}))
				return
			}
			if ownerOnly && !rbac.GroupRole(role).IsOwner() {
				httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(httpx.CodeForbidden, "仅群主可操作", httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonNotGroupOwner}))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

const maxGuardBodyBytes = 64 << 10

// guardGroupID 从请求体提取 group_id（守卫子树统一入口参数）。
// 失败时返回原始 request（body 已还原，供 handler 正常解析并返回 10000）。
func guardGroupID(r *http.Request) (string, bool) {
	if r.Body == nil {
		return "", false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 || len(body) > maxGuardBodyBytes {
		return "", false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	raw, ok := payload["group_id"]
	if !ok {
		return "", false
	}
	var groupID string
	if err := json.Unmarshal(raw, &groupID); err != nil {
		return "", false
	}
	return strings.TrimSpace(groupID), true
}
