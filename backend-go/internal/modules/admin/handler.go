package admin

import (
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
	var req AdminRbacRoleCreateRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	payload, apiErr, err := h.service.CreateRole(withRequest(r.Context(), r), req)
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

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRoleUpdateRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	var req AdminRbacGrantAdminRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.GrantAdmin(withRequest(r.Context(), r), req)
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

func (h *Handler) RevokeAdmin(w http.ResponseWriter, r *http.Request) {
	var req AdminRbacRevokeAdminRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	var req AdminBanUserRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.BanUser(withRequest(r.Context(), r), req)
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

func (h *Handler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	var req AdminUnbanUserRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UnbanUser(withRequest(r.Context(), r), req)
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	var req AdminDismissGroupRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DismissGroup(r.Context(), req)
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

func (h *Handler) SearchReports(w http.ResponseWriter, r *http.Request) {
	var req AdminReportListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	var req AdminReportHandleRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.HandleReport(r.Context(), req)
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

func (h *Handler) UpdateAppConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateAppConfigRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.UpdateAppConfig(withRequest(r.Context(), r), req)
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	apiErr, err := h.service.DeleteTag(withRequest(r.Context(), r), req.Name, req.HardDelete)
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		h.writeAPIError(w, r, httpx.InvalidRequest("参数错误"))
		return
	}
	ctx := withRequest(r.Context(), r)
	idempotency.ReplayOrStore(w, r, "admin/announcements/create", func() httpx.ApiResponse[AnnouncementCreated] {
		payload, apiErr, err := h.service.CreateAnnouncement(ctx, req)
		if err != nil {
			observability.L().Error("handler error", "method", "CreateAnnouncement", "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
			return httpx.ApiResponse[AnnouncementCreated]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
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
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
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
