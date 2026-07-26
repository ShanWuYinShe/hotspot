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

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req NotificationListRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
		list, err := h.service.List(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrInvalidNotificationRequest) {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
				return
			}
			logServerError(r, "List", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
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
			logServerError(r, "UnreadCount", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
			return
		}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(count))
}

// MarkRead is the unified mark_read endpoint (api_spec_notify.md). It replaces
// the legacy mark_as_read and mark_all_read endpoints. The scope field selects
// the operation: "all" clears every unread notification; "selected" (the
// default) marks either the supplied notification_ids or all notifications of
// the supplied type as read (exactly one of the two must be provided).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	scope := req.Scope
	if scope == "" {
		scope = "selected"
	}
	switch scope {
	case "all":
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			logServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	case "selected":
		hasIDs := len(req.NotificationIDs) > 0
		hasType := req.Type != ""
		if hasIDs == hasType {
			// exactly one of notification_ids / type required
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		if hasIDs {
			if len(req.NotificationIDs) > 50 {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
				return
			}
			if err := h.service.MarkAsRead(r.Context(), req.NotificationIDs); err != nil {
				logServerError(r, "MarkRead", err)
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
				return
			}
			httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
			return
		}
		// scope=selected + type: mark all unread of that type. Currently the
		// service exposes MarkAsRead (by ids) and MarkAllRead (everything) only;
		// emulate "by type" by passing the type through the repo via the
		// MarkAllReadByType helper when available. The current Repository
		// interface lacks a by-type method, so fall through to MarkAllRead only
		// when type filtering is not supported. For now we treat type as
		// accepted and clear all unread of the requested type via the typed
		// service method when present.
		if err := h.service.MarkAllReadByType(r.Context(), req.Type); err != nil {
			logServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
	}
}
