package private_messages

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
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

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		logBusinessError(r, "Send", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req SendPrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Send", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	if !isValidSendRequest(req) {
		logBusinessError(r, "Send", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "private_messages/send", func() httpx.ApiResponse[MessageSent] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			logServerError(r, "Send", err)
			return httpx.ApiResponse[MessageSent]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(msg)
	})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	var req GetHistoryRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "GetHistory", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	if req.TargetUserID == "" || !isValidPagination(req.Pagination) {
		logBusinessError(r, "GetHistory", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	history, err := h.service.History(r.Context(), req)
	if err != nil {
		logServerError(r, "GetHistory", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
		if err := idempotency.RequireKey(r); err != nil {
			logBusinessError(r, "Update", err.Code, err.Message)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
			return
		}
		var req UpdatePrivateMessageRequest
		if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" || req.Content == "" {
			logBusinessError(r, "Update", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		idempotency.ReplayOrStore(w, r, "private_messages/update", func() httpx.ApiResponse[struct{}] {
			if err := h.service.Update(r.Context(), req); err != nil {
				switch {
				case errors.Is(err, ErrMessageNotFound):
					logBusinessError(r, "Update", 10300, "消息不存在")
					return httpx.ApiResponse[struct{}]{Code: 10300, Message: "消息不存在"}
				case errors.Is(err, ErrMessagePermissionDenied):
					logBusinessError(r, "Update", 10200, "无权限编辑消息")
					return httpx.ApiResponse[struct{}]{Code: 10200, Message: "无权限编辑消息"}
				case errors.Is(err, ErrMessageNotEditable):
					logBusinessError(r, "Update", 10400, "消息不可编辑")
					return httpx.ApiResponse[struct{}]{Code: 10400, Message: "消息不可编辑"}
				default:
					logServerError(r, "Update", err)
					return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
				}
			}
			return httpx.SuccessEmpty()
		})
	}

func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
		if err := idempotency.RequireKey(r); err != nil {
			logBusinessError(r, "Recall", err.Code, err.Message)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
			return
		}
		var req RecallPrivateMessageRequest
		if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" {
			logBusinessError(r, "Recall", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		idempotency.ReplayOrStore(w, r, "private_messages/recall", func() httpx.ApiResponse[struct{}] {
			if err := h.service.Recall(r.Context(), req); err != nil {
				switch {
				case errors.Is(err, ErrMessageNotFound):
					logBusinessError(r, "Recall", 10300, "消息不存在")
					return httpx.ApiResponse[struct{}]{Code: 10300, Message: "消息不存在"}
				case errors.Is(err, ErrMessagePermissionDenied):
					logBusinessError(r, "Recall", 10200, "无权限撤回消息")
					return httpx.ApiResponse[struct{}]{Code: 10200, Message: "无权限撤回消息"}
				case errors.Is(err, ErrMessageNotRecallable):
					logBusinessError(r, "Recall", 10400, "消息不可撤回")
					return httpx.ApiResponse[struct{}]{Code: 10400, Message: "消息不可撤回"}
				default:
					logServerError(r, "Recall", err)
					return httpx.ApiResponse[struct{}]{Code: 10600, Message: "服务暂时不可用"}
				}
			}
			return httpx.SuccessEmpty()
		})
	}

// MarkRead is the unified mark_read endpoint for private messages. It replaces
// the legacy /me/mark_as_read and /me/mark_all_read endpoints. scope selects
// the operation: "all" clears every unread private message for the caller;
// "session" (the default) marks messages from target_user_id as read (up to
// the optional up_to_message_id).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkPrivateReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "MarkRead", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	scope := req.Scope
	if scope == "" {
		scope = "session"
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
	case "session":
		if req.TargetUserID == "" {
			logBusinessError(r, "MarkRead", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		if err := h.service.MarkAsRead(r.Context(), MarkPrivateMessagesReadRequest{
			TargetUserID:  req.TargetUserID,
			UpToMessageID: req.UpToMessageID,
		}); err != nil {
			if errors.Is(err, ErrMessageNotFound) {
				logBusinessError(r, "MarkRead", 10300, "消息不存在")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
				return
			}
			logServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		logBusinessError(r, "MarkRead", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
	}
}

func isValidSendRequest(req SendPrivateMessageRequest) bool {
	if req.TargetUserID == "" || req.Type == "" {
		return false
	}
	switch req.Type {
	case "text":
		return req.Content != ""
	case "image", "video", "audio", "file":
		return req.FileID != ""
	case "share":
		return req.ShareCode != ""
	default:
		return false
	}
}

func isValidPagination(p *PaginationParams) bool {
	if p == nil {
		return true
	}
	if p.Limit < 0 || p.Limit > 100 {
		return false
	}
	if p.Cursor == "" {
		return true
	}
	_, _, err := decodeMessageCursor(p.Cursor)
	return err == nil
}
