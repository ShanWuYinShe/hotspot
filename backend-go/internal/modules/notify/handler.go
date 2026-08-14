package notify

import (
	"errors"
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
	var req NotifyListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	list, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidNotifyRequest) {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		httpx.WriteAPIError(w, r, "List", err)
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
		httpx.WriteAPIError(w, r, "UnreadCount", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(count))
}

// MarkRead marks notifys as read (spec api_spec_notify.md).
// scope='ids' requires notify_ids (1-50); scope='all' marks everything read and
// ignores notify_ids. scope is mandatory — a missing scope is invalid (10000).
// Unknown or foreign ids are silently ignored by the repository.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	switch MarkReadScope(req.Scope) {
	case MarkReadScopeIDs:
		if len(req.NotifyIDs) == 0 || len(req.NotifyIDs) > maxMarkReadIDs {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		if err := h.service.MarkAsRead(r.Context(), req.NotifyIDs); err != nil {
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
	case MarkReadScopeAll:
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
	default:
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
