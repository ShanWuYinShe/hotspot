package notify

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req NotificationListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidNotificationRequest) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		observability.LogServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(list))
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	count, err := h.service.UnreadCount(r.Context())
	if err != nil {
		observability.LogServerError(r, "UnreadCount", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(count))
}

// MarkRead marks notifications as read (spec api_spec_notify.md). notify_ids
// selects 1-50 notifications; absent or empty means "mark everything read".
// Unknown or foreign ids are silently ignored by the repository.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if len(req.NotifyIDs) == 0 {
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			observability.LogServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	}
	if len(req.NotifyIDs) > 50 {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.MarkAsRead(r.Context(), req.NotifyIDs); err != nil {
		observability.LogServerError(r, "MarkRead", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
