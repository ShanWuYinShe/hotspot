package privatemessages

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

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		observability.LogBusinessError(r, "Send", int(err.Code), err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req SendPrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "Send", int(err.Code), err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	if !isValidSendRequest(req) {
		observability.LogBusinessError(r, "Send", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "private_messages/send", func() httpx.ApiResponse[any] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			var blocked *MessageBlockedError
			if errors.As(err, &blocked) {
				// 契约(api_spec_chat.md):发送被拒绝返回 httpx.CodeForbidden,data 携带 { reason: BlockReason }
				observability.LogBusinessError(r, "Send", int(httpx.CodeForbidden), "发送被拒绝")
				var data any = map[string]string{"reason": string(blocked.Reason)}
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "发送被拒绝", Data: &data}
			}
			observability.LogServerError(r, "Send", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = msg
		return httpx.ApiResponse[any]{Code: 0, Data: &data}
	})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	var req GetHistoryRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "GetHistory", int(err.Code), err.Message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	if req.TargetUserID == "" || !isValidPagination(req.Pagination) {
		observability.LogBusinessError(r, "GetHistory", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	history, err := h.service.History(r.Context(), req)
	if err != nil {
		observability.LogServerError(r, "GetHistory", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdatePrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" || req.Content == "" {
		observability.LogBusinessError(r, "Update", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			observability.LogBusinessError(r, "Update", int(httpx.CodeNotFound), "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
		case errors.Is(err, ErrMessagePermissionDenied):
			observability.LogBusinessError(r, "Update", int(httpx.CodeForbidden), "无权限编辑消息")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "无权限编辑消息"))
		case errors.Is(err, ErrMessageNotEditable):
			observability.LogBusinessError(r, "Update", int(httpx.CodeConflict), "消息不可编辑")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "消息不可编辑"))
		default:
			observability.LogServerError(r, "Update", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	var req RecallPrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" {
		observability.LogBusinessError(r, "Recall", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeNotFound), "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
		case errors.Is(err, ErrMessagePermissionDenied):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeForbidden), "无权限撤回消息")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "无权限撤回消息"))
		case errors.Is(err, ErrMessageNotRecallable):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeConflict), "消息不可撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "消息不可撤回"))
		default:
			observability.LogServerError(r, "Recall", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

// MarkRead is the unified mark_read endpoint for private messages. It replaces
// the legacy /me/mark_as_read and /me/mark_all_read endpoints. scope selects
// the operation: "all" clears every unread private message for the caller;
// "session" (the default) marks messages from target_user_id as read (up to
// the optional up_to_message_id).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkPrivateReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		observability.LogBusinessError(r, "MarkRead", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	scope := req.Scope
	if scope == "" {
		scope = "session"
	}
	switch scope {
	case "all":
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			observability.LogServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	case "session":
		if req.TargetUserID == "" {
			observability.LogBusinessError(r, "MarkRead", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		if err := h.service.MarkAsRead(r.Context(), MarkPrivateMessagesReadRequest{
			TargetUserID:  req.TargetUserID,
			UpToMessageID: req.UpToMessageID,
		}); err != nil {
			if errors.Is(err, ErrMessageNotFound) {
				observability.LogBusinessError(r, "MarkRead", int(httpx.CodeNotFound), "消息不存在")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
				return
			}
			observability.LogServerError(r, "MarkRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		observability.LogBusinessError(r, "MarkRead", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
