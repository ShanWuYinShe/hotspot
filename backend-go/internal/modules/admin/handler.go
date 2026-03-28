package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	payload, apiErr, err := h.service.GetMe(r.Context())
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	payload, apiErr, err := h.service.ListPermissions(r.Context())
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	payload, apiErr, err := h.service.ListRoles(r.Context())
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req AdminRbacRoleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/rbac/roles/create", func() httpx.ApiResponse[AdminRoleItem] {
		payload, apiErr, err := h.service.CreateRole(withRequest(r.Context(), r), req)
		if err != nil {
			observability.L().Error("handler error", "method", "CreateRole", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[AdminRoleItem]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "CreateRole", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[AdminRoleItem]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.Success(payload)
	})
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateRole(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteRole(withRequest(r.Context(), r), req.RoleID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleSetPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.SetRolePermissions(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacAdminsListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.ListAdmins(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) GrantAdmin(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req AdminRbacGrantAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/rbac/admins/grant", func() httpx.ApiResponse[struct{}] {
		apiErr, err := h.service.GrantAdmin(withRequest(r.Context(), r), req)
		if err != nil {
			observability.L().Error("handler error", "method", "GrantAdmin", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "GrantAdmin", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) RevokeAdmin(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRevokeAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.RevokeAdmin(withRequest(r.Context(), r), req.UserID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	var req AuditLogListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.ListAuditLogs(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) OpenAuditLog(w http.ResponseWriter, r *http.Request) {
	var req AuditLogOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.GetAuditLog(r.Context(), req.LogID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	var req AdminUserSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchUsers(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) OpenAdminUser(w http.ResponseWriter, r *http.Request) {
	var req AdminUserOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.OpenAdminUser(r.Context(), req.UserID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) BanUser(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req AdminBanUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/users/ban", func() httpx.ApiResponse[struct{}] {
		apiErr, err := h.service.BanUser(r.Context(), req)
		if err != nil {
			observability.L().Error("handler error", "method", "BanUser", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "BanUser", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	var req AdminUnbanUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UnbanUser(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SearchGroups(w http.ResponseWriter, r *http.Request) {
	var req AdminGroupSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchGroups(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) OpenAdminGroup(w http.ResponseWriter, r *http.Request) {
	var req AdminGroupOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.OpenAdminGroup(r.Context(), req.GroupID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) DismissGroup(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req AdminDismissGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/groups/dismiss", func() httpx.ApiResponse[struct{}] {
		apiErr, err := h.service.DismissGroup(r.Context(), req)
		if err != nil {
			observability.L().Error("handler error", "method", "DismissGroup", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "DismissGroup", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) SearchReports(w http.ResponseWriter, r *http.Request) {
	var req AdminReportSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchReports(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) HandleReport(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req AdminReportHandleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "admin/reports/handle", func() httpx.ApiResponse[struct{}] {
		apiErr, err := h.service.HandleReport(r.Context(), req)
		if err != nil {
			observability.L().Error("handler error", "method", "HandleReport", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "HandleReport", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[struct{}]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.SuccessEmpty()
	})
}

func (h *Handler) GetAppConfig(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	payload, apiErr, err := h.service.GetAppConfig(r.Context())
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) UpdateUpload(w http.ResponseWriter, r *http.Request) {
	var req UpdateUploadConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateUpload(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateDefaults(w http.ResponseWriter, r *http.Request) {
	var req UpdateDefaultsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateDefaults(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateRateLimits(w http.ResponseWriter, r *http.Request) {
	var req UpdateRateLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateRateLimits(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateResponseDefaults(w http.ResponseWriter, r *http.Request) {
	var req UpdateResponseDefaultsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateResponseDefaults(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateFeatures(w http.ResponseWriter, r *http.Request) {
	var req UpdateFeaturesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateFeatures(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateLimits(w http.ResponseWriter, r *http.Request) {
	var req UpdateLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateLimits(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	var req UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateContact(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SearchTags(w http.ResponseWriter, r *http.Request) {
	var req AdminTagSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchTags(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	var req AdminDeleteTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteTag(withRequest(r.Context(), r), req.Name)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) SearchAnnouncements(w http.ResponseWriter, r *http.Request) {
	var req AdminAnnouncementSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.SearchAnnouncements(r.Context(), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		h.writeAPIError(w, r, err)
		return
	}
	var req CreateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	ctx := withRequest(r.Context(), r)
	idempotency.ReplayOrStore(w, r, "admin/announcements/create", func() httpx.ApiResponse[AnnouncementCreated] {
		payload, apiErr, err := h.service.CreateAnnouncement(ctx, req)
		if err != nil {
			observability.L().Error("handler error", "method", "CreateAnnouncement", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[AnnouncementCreated]{Code: 10600, Message: "服务暂时不可用"}
		}
		if apiErr != nil {
			observability.L().Warn("handler business error", "method", "CreateAnnouncement", "code", apiErr.Code, "message", apiErr.Message, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[AnnouncementCreated]{Code: apiErr.Code, Message: apiErr.Message}
		}
		return httpx.Success(payload)
	})
}

func (h *Handler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req UpdateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateAnnouncement(withRequest(r.Context(), r), req)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	var req DeleteAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteAnnouncement(withRequest(r.Context(), r), req.AnnouncementID)
	if err != nil {
		h.writeUnavailable(w, r)
		return
	}
	if apiErr != nil {
		h.writeAPIError(w, r, apiErr)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeAPIError(w http.ResponseWriter, r *http.Request, err *httpx.APIError) {
	observability.L().Warn("handler business error", "method", handlerMethodName(r), "code", err.Code, "message", err.Message, "request_id", observability.RequestIDFromContext(r.Context()))
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
}

func (h *Handler) writeUnavailable(w http.ResponseWriter, r *http.Request) {
	observability.L().Error("handler error", "method", handlerMethodName(r), "error", errHandlerUnavailable, "request_id", observability.RequestIDFromContext(r.Context()))
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
}

var errHandlerUnavailable = errors.New("服务暂时不可用")

func handlerMethodName(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "unknown"
	}
	pattern := rctx.RoutePattern()
	if pattern == "" {
		return "unknown"
	}
	return r.Method + " " + pattern
}
