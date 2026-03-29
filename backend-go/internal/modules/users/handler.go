package users

import (
	"encoding/json"
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	profile, err := h.service.GetMyProfile(r.Context())
	if err != nil {
		h.writeServiceError(w, r, "GetMyProfile", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(profile))
}

func (h *Handler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if err := h.service.UpdateMyProfile(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "UpdateMyProfile", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) GetMySettings(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	settings, err := h.service.GetMySettings(r.Context())
	if err != nil {
		h.writeServiceError(w, r, "GetMySettings", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(settings))
}

func (h *Handler) UpdateMySettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if err := h.service.UpdateMySettings(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "UpdateMySettings", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, method string, err error) {
	observability.L().Error("handler error",
		"method", method,
		"error", err,
		"request_id", observability.RequestIDFromContext(r.Context()),
	)
	switch {
	case errors.Is(err, ErrUnauthorized):
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10100, "认证失败"))
	case errors.Is(err, ErrPermissionDenied):
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
	case errors.Is(err, ErrUserNotFound):
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "用户不存在"))
	case errors.Is(err, ErrConflict):
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "业务冲突"))
	case errors.Is(err, ErrInvalidCursor):
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "分页游标无效"))
	default:
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
	}
}
