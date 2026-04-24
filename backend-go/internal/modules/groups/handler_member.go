package groups

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	var req GroupApplicationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListApplications(r.Context(), req)
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

func (h *Handler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req ApproveGroupApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/applications/approve", func() httpx.ApiResponse[struct{}] {
		if err := h.service.ApproveApplication(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyProcessed {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "申请已处理"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req RejectGroupApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/applications/reject", func() httpx.ApiResponse[struct{}] {
		if err := h.service.RejectApplication(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			if err == ErrAlreadyProcessed {
				return httpx.ApiResponse[struct{}]{Code: 10400, Message: "申请已处理"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	var req SetMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" || req.Role == "" || (req.Role != GroupRoleAdmin && req.Role != GroupRoleMember) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.SetMemberRole(r.Context(), req); err != nil {
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

func (h *Handler) MuteMember(w http.ResponseWriter, r *http.Request) {
	var req MuteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.MuteMember(r.Context(), req); err != nil {
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

func (h *Handler) UnmuteMember(w http.ResponseWriter, r *http.Request) {
	var req UnmuteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UnmuteMember(r.Context(), req); err != nil {
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

func (h *Handler) BanMember(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req BanMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/members/ban", func() httpx.ApiResponse[struct{}] {
		if err := h.service.BanMember(r.Context(), req); err != nil {
			if err == ErrGroupNotFound {
				return httpx.ApiResponse[struct{}]{Code: 10300, Message: "群组不存在"}
			}
			if err == ErrAccessRestricted {
				return httpx.ApiResponse[struct{}]{Code: 10200, Message: "权限不足"}
			}
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(struct{}{})
	})
}

func (h *Handler) UnbanMember(w http.ResponseWriter, r *http.Request) {
	var req UnbanMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.UnbanMember(r.Context(), req); err != nil {
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

func (h *Handler) ListBannedMembers(w http.ResponseWriter, r *http.Request) {
	var req BannedMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	items, nextCursor, err := h.service.ListBannedMembers(r.Context(), req)
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

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	var req RemoveMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.UserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.RemoveMember(r.Context(), req); err != nil {
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

func (h *Handler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	var req TransferOwnershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.NewOwnerID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.TransferOwnership(r.Context(), req); err != nil {
		if err == ErrGroupNotFound {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
			return
		}
		if err == ErrAccessRestricted {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
			return
		}
		if err == ErrTargetNotMember {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "目标用户不是群成员"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}
