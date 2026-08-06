package users

import (
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

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code httpx.Code, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
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
	var req UpdateAccountRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
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
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	if err := h.service.UpdateMySettings(r.Context(), req); err != nil {
		h.writeServiceError(w, r, "UpdateMySettings", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, method string, err error) {
	apiErr := apiErrorFromService(err)
	if apiErr.Code == httpx.CodeInternalUnavailable {
		logServerError(r, method, err)
	} else {
		logBusinessError(r, method, apiErr.Code, apiErr.Message)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
}

func apiErrorFromService(err error) *httpx.APIError {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "认证失败"}
	case errors.Is(err, ErrPermissionDenied):
		return &httpx.APIError{Code: httpx.CodeForbidden, Message: "权限不足"}
	case errors.Is(err, ErrUserNotFound):
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	case errors.Is(err, ErrConflict):
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "业务冲突"}
	case errors.Is(err, ErrInvalidCursor):
		return &httpx.APIError{Code: httpx.CodeInvalidRequest, Message: "分页游标无效"}
	default:
		return &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
	}
}

func errorResponse[T any](err error) httpx.ApiResponse[T] {
	apiErr := apiErrorFromService(err)
	return httpx.ApiResponse[T]{Code: apiErr.Code, Message: apiErr.Message}
}
