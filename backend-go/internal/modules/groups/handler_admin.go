package groups

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

// GetSettings handles POST /api/v1/groups/admin/settings（守卫子树）。
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var req GetGroupSettingsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	settings, ok, err := h.service.GetSettings(r.Context(), req.GroupID)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.admin.get_settings", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(settings))
}

// ListAdminMembers handles POST /api/v1/groups/admin/members/list（守卫子树）。
func (h *Handler) ListAdminMembers(w http.ResponseWriter, r *http.Request) {
	var req ListAdminGroupMembersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListAdminMembers(r.Context(), req)
	if err != nil {
		if _, ok := err.(*httpx.APIError); ok {
			httpx.WriteAPIError(w, r, "groups.admin.members.list", err)
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "群组不存在"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

// RecallGroupMessageAsAdmin handles POST /api/v1/groups/admin/messages/recall（守卫子树）。
func (h *Handler) RecallGroupMessageAsAdmin(w http.ResponseWriter, r *http.Request) {
	var req RecallGroupMessageAdminRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.GroupID == "" || req.MessageID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.RecallGroupMessageAsAdmin(r.Context(), req); err != nil {
		httpx.WriteAPIError(w, r, "groups.admin.messages.recall", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
