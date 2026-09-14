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
		observability.LogServerError(r, method, err)
	} else {
		observability.LogBusinessError(r, method, int(apiErr.Code), apiErr.Message)
	}
	if apiErr.Data != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.ErrorWithData(apiErr.Code, apiErr.Message, apiErr.Data))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(apiErr.Code, apiErr.Message))
}

func apiErrorFromService(err error) *httpx.APIError {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "认证失败"}
	case errors.Is(err, ErrPermissionDenied):
		// 10200 一律通过 httpx.Forbidden 构造，携带 data.forbidden_reason（spec api_spec_errors.md）。
		// users 模块内 ErrPermissionDenied 均为对方隐私设置拒绝（allow_follow/social_lists_visible/groups_visible/reacted_moments_visible）。
		return httpx.Forbidden(httpx.ReasonPrivacySetting, "对方隐私设置不允许")
	case errors.Is(err, ErrUserNotFound):
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	case errors.Is(err, ErrBlockedTag):
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "标签包含被屏蔽内容"}
	case errors.Is(err, ErrLoginIDTaken):
		// 10400（spec api_spec_user.md → upgrade：login_id 已被占用）
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "login_id 已被占用"}
	case errors.Is(err, ErrAvatarNotOwned):
		// 10300（spec api_spec_user.md → update_account：avatar_file_id 对应的
		// 媒体资源不存在或不属于当前用户）
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "媒体资源不存在或不属于当前用户"}
	case errors.Is(err, ErrConflict):
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "业务冲突"}
	case errors.Is(err, ErrInvalidRequest):
		return &httpx.APIError{Code: httpx.CodeInvalidRequest, Message: "参数错误"}
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
