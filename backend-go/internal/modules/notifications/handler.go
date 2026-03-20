package notifications

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req NotificationListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.List(r.Context(), req)))
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.UnreadCount(r.Context())))
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	var req MarkAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.NotificationIDs) == 0 || len(req.NotificationIDs) > 100 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	h.service.MarkAsRead(r.Context(), req.NotificationIDs)
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	h.service.MarkAllRead(r.Context())
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
