package privatechats

import (
	"encoding/json"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		observability.L().Error("handler error",
			"method", "List",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req OpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUserID == "" || (req.IsMuted == nil && req.IsPinned == nil) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Delete(r.Context(), req.TargetUserID); err != nil {
		h.handleServiceError(w, r, "Delete", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, method string, err error) {
	observability.L().Error("handler error",
		"method", method,
		"error", err,
		"request_id", observability.RequestIDFromContext(r.Context()),
	)
	if errors.Is(err, users.ErrUserNotFound) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "用户不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
}
