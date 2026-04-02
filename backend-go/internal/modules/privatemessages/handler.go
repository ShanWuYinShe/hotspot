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
	var req UpdatePrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" || req.Content == "" {
		logBusinessError(r, "Update", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			logBusinessError(r, "Update", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrMessagePermissionDenied):
			logBusinessError(r, "Update", 10200, "无权限编辑消息")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "无权限编辑消息"))
		case errors.Is(err, ErrMessageNotEditable):
			logBusinessError(r, "Update", 10400, "消息不可编辑")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息不可编辑"))
		default:
			logServerError(r, "Update", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	var req RecallPrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" {
		logBusinessError(r, "Recall", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			logBusinessError(r, "Recall", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrMessagePermissionDenied):
			logBusinessError(r, "Recall", 10200, "无权限撤回消息")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "无权限撤回消息"))
		case errors.Is(err, ErrMessageNotRecallable):
			logBusinessError(r, "Recall", 10400, "消息不可撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息不可撤回"))
		default:
			logServerError(r, "Recall", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	var req MarkPrivateMessagesReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetUserID == "" {
		logBusinessError(r, "MarkAsRead", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.MarkAsRead(r.Context(), req); err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			logBusinessError(r, "MarkAsRead", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
			return
		}
		logServerError(r, "MarkAsRead", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		logBusinessError(r, "MarkAllRead", err.Code, err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	if err := h.service.MarkAllRead(r.Context()); err != nil {
		logServerError(r, "MarkAllRead", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
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
	return true
}
