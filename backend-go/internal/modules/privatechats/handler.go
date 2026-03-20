package privatechats

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service   *Service
	appConfig appconfig.Source
}

func NewHandler(service *Service, appConfig appconfig.Source) *Handler {
	return &Handler{service: service, appConfig: appConfig}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.List(r.Context())))
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req OpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetUserID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	cfg := h.appConfig.Get(r.Context())
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.OpenTheDoor(r.Context(), req.TargetUserID, cfg.ResponseDefaults.ChatRecentMessagesCount)))
}
