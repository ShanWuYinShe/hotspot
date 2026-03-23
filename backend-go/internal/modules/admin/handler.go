package admin

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	payload, apiErr, err := h.service.GetMe(r.Context())
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	payload, apiErr, err := h.service.ListPermissions(r.Context())
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	payload, apiErr, err := h.service.ListRoles(r.Context())
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	var req AdminRbacRoleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/rbac/roles/create", func() httpx.ApiResponse[AdminRoleItem] {
		payload, apiErr, err := h.service.CreateRole(r.Context(), req)
		if err != nil {
			return httpx.ApiResponse[AdminRoleItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			return httpx.ApiResponse[AdminRoleItem]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.Success(payload)
	})
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateRole(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteRole(r.Context(), req.RoleID)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleSetPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.SetRolePermissions(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacAdminsListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.ListAdmins(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) GrantAdmin(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	var req AdminRbacGrantAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/rbac/admins/grant", func() httpx.ApiResponse[struct{}] {
		apiErr, err := h.service.GrantAdmin(r.Context(), req)
		if err != nil {
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) RevokeAdmin(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRevokeAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.RevokeAdmin(r.Context(), req.UserID)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) GetAppConfig(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	payload, apiErr, err := h.service.GetAppConfig(r.Context())
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) SearchTags(w http.ResponseWriter, r *http.Request) {
	var req AdminTagSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchTags(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	var req AdminDeleteTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteTag(r.Context(), req.Name)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SearchAnnouncements(w http.ResponseWriter, r *http.Request) {
	var req AdminAnnouncementSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchAnnouncements(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, err)
		return
	}
	var req CreateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/announcements/create", func() httpx.ApiResponse[AnnouncementCreated] {
		payload, apiErr, err := h.service.CreateAnnouncement(r.Context(), req)
		if err != nil {
			return httpx.ApiResponse[AnnouncementCreated]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			return httpx.ApiResponse[AnnouncementCreated]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.Success(payload)
	})
}

func (h *Handler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req UpdateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateAnnouncement(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req DeleteAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteAnnouncement(r.Context(), req.AnnouncementID)
	if err != nil {
		h.writeUnavailable(w)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeAPIError(w http.ResponseWriter, err *httpx.APIError) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
}

func (h *Handler) writeUnavailable(w http.ResponseWriter) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
}
