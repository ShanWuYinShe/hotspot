package privatemessages

import (
	"errors"
	"net/http"
	"unicode/utf8"

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
		httpx.WriteAPIError(w, r, "Send", err)
		return
	}
	var req SendPrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteAPIError(w, r, "Send", err)
		return
	}
	if !isValidSendRequest(req) {
		httpx.WriteInvalidRequest(w, r, "Send", "参数错误")
		return
	}
	idempotency.ReplayOrStore(w, r, "private_messages/send", func() httpx.ApiResponse[any] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			var apiErr *httpx.APIError
			if errors.As(err, &apiErr) {
				// 契约(api_spec_errors.md):10200 必带 data.forbidden_reason(httpx.Forbidden 自动带出)
				observability.LogBusinessError(r, "Send", int(apiErr.Code), apiErr.Message)
				if apiErr.Data != nil {
					var data any = apiErr.Data
					return httpx.ApiResponse[any]{Code: apiErr.Code, Message: apiErr.Message, Data: &data}
				}
				return httpx.ApiResponse[any]{Code: apiErr.Code, Message: apiErr.Message}
			}
			// 契约(api_spec_chat.md → private_messages/send 10400)：share_code 无效等业务冲突
			if errors.Is(err, ErrInvalidShareCode) {
				return httpx.Business[any](r, "Send", httpx.CodeConflict, "分享链接无效或已过期")
			}
			observability.LogServerError(r, "Send", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = msg
		return httpx.ApiResponse[any]{Code: 0, Data: &data}
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListPrivateMessagesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteAPIError(w, r, "List", err)
		return
	}
	if req.TargetUserID == "" || !isValidPagination(req.Pagination) {
		httpx.WriteInvalidRequest(w, r, "List", "参数错误")
		return
	}
	history, err := h.service.History(r.Context(), req)
	if err != nil {
		httpx.WriteAPIError(w, r, "List", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdatePrivateMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.MessageID == "" || req.Content == "" {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误")
		return
	}
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeNotFound, "消息不存在")
		case errors.Is(err, ErrMessagePermissionDenied):
			// 契约(api_spec_chat.md → private_messages/update)：非作者编辑 → 10200 + not_resource_owner
			httpx.WriteAPIError(w, r, "Update", httpx.Forbidden(httpx.ReasonNotResourceOwner, "无权限编辑消息"))
		case errors.Is(err, ErrMessageNotEditable):
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeConflict, "消息不可编辑")
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
		httpx.WriteInvalidRequest(w, r, "Recall", "参数错误")
		return
	}
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteBusinessError(w, r, "Recall", httpx.CodeNotFound, "消息不存在")
		case errors.Is(err, ErrMessagePermissionDenied):
			// 契约(api_spec_chat.md → private_messages/recall)：非作者撤回 → 10200 + not_resource_owner
			httpx.WriteAPIError(w, r, "Recall", httpx.Forbidden(httpx.ReasonNotResourceOwner, "无权限撤回消息"))
		case errors.Is(err, ErrMessageNotRecallable):
			httpx.WriteBusinessError(w, r, "Recall", httpx.CodeConflict, "消息不可撤回")
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
// "session" marks messages from target_user_id as read (up to the optional
// up_to_message_id). scope is required (spec 参数必显式).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkPrivateReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "参数错误")
		return
	}
	scope := req.Scope
	// 参数必显式（spec api_spec_base.md）：scope 影响行为范围，缺省不得静默翻转语义
	if scope == "" {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "scope 必传")
		return
	}
	// 参数互斥（spec chat.md → mark_read）：scope='all' 时不得传 up_to_message_id
	if scope == "all" && req.UpToMessageID != "" {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "scope=all 时不得传 up_to_message_id")
		return
	}
	switch scope {
	case "all":
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	case "session":
		if req.TargetUserID == "" {
			httpx.WriteInvalidRequest(w, r, "MarkRead", "参数错误")
			return
		}
		if err := h.service.MarkAsRead(r.Context(), MarkPrivateMessagesReadRequest{
			TargetUserID:  req.TargetUserID,
			UpToMessageID: req.UpToMessageID,
		}); err != nil {
			if errors.Is(err, ErrMessageNotFound) {
				httpx.WriteBusinessError(w, r, "MarkRead", httpx.CodeNotFound, "消息不存在")
				return
			}
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		httpx.WriteBusinessError(w, r, "MarkRead", httpx.CodeInvalidRequest, "参数错误")
	}
}

// messageContentMaxRunes 契约级上限（api_spec_base.md 字段校验约束：content(message) 1-5000 字符）。
const messageContentMaxRunes = 5000

func isValidSendRequest(req SendPrivateMessageRequest) bool {
	if req.TargetUserID == "" || req.Type == "" {
		return false
	}
	switch req.Type {
	case "text":
		return req.Content != "" && utf8.RuneCountInString(req.Content) <= messageContentMaxRunes
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
