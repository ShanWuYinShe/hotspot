package privatechats

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service   *Service
	appConfig appconfig.Source
}

func NewHandler(service *Service, appConfig appconfig.Source) *Handler {
	return &Handler{service: service, appConfig: appConfig}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req MyPrivateChatsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		observability.LogServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req OpenTheDoorRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetUserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	cfg := h.appConfig.Get(r.Context())
	result, err := h.service.OpenTheDoor(r.Context(), req.TargetUserID, cfg.ResponseDefaults.ChatRecentMessagesCount)
	if err != nil {
		h.handleServiceError(w, r, "OpenTheDoor", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdatePrivateChatSettingsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetUserID == "" || (req.IsMuted == nil && req.IsPinned == nil) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.UpdateSettings(r.Context(), req); err != nil {
		h.handleServiceError(w, r, "UpdateSettings", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeletePrivateChatRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetUserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Delete(r.Context(), req.TargetUserID); err != nil {
		h.handleServiceError(w, r, "Delete", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, method string, err error) {
	if errors.Is(err, users.ErrUserNotFound) {
		observability.LogBusinessError(r, method, int(httpx.CodeNotFound), "用户不存在")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "用户不存在"))
		return
	}
	observability.LogServerError(r, method, err)
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
}
