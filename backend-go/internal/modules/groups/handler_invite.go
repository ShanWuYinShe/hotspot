package groups

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func (h *Handler) ListMyInvites(w http.ResponseWriter, r *http.Request) {
	var req MyInvitesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	items, nextCursor, err := h.service.ListMyInvites(r.Context(), req)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/me/invites/accept", func() httpx.ApiResponse[struct{}] {
		if err := h.service.AcceptInvite(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RejectInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/me/invites/reject", func() httpx.ApiResponse[struct{}] {
		if err := h.service.RejectInvite(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrInviteExpired {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "邀请已失效"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/members/invite", func() httpx.ApiResponse[Invitation] {
		item, err := h.service.InviteMember(r.Context(), req)
		if err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[Invitation]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[Invitation]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyMember {
				return httpx.ApiResponse[Invitation]{Code: 10400, Message: "已是群成员"}
			}
			return httpx.ApiResponse[Invitation]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(item)
	})
}

func (h *Handler) ListSentInvites(w http.ResponseWriter, r *http.Request) {
	var req GroupSentInvitesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListSentInvites(r.Context(), req)
	if err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	resp := map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	var req RevokeSentInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.InviteID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.RevokeInvite(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
