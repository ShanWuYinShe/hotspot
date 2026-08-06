package groupmessages

import (
	"errors"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteAPIError(w, r, "Send", err)
		return
	}
	var req SendGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Send", "请求体格式错误")
		return
	}
	if req.GroupID == "" || req.Type == "" {
		httpx.WriteInvalidRequest(w, r, "Send", "参数错误")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.SenderUserID = principal.UserID
	idempotency.ReplayOrStore(w, r, "group_messages/send", func() httpx.ApiResponse[any] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrNotGroupMember) {
				observability.LogBusinessError(r, "Send", int(httpx.CodeConflict), "不是群组成员")
				return httpx.ApiResponse[any]{Code: httpx.CodeConflict, Message: "不是群组成员"}
			}
			if errors.Is(err, ErrUserMuted) {
				mutedUntil := ""
				var mutedErr *MutedError
				if errors.As(err, &mutedErr) {
					mutedUntil = mutedErr.MutedUntil
				}
				// 契约(api_spec_chat.md):禁言拒绝返回 10400,data 携带 { mute: { muted_until } }
				// 结构化携带截止时间,避免客户端从 message 文本解析时间戳。
				observability.LogBusinessError(r, "Send", int(httpx.CodeConflict), "您已被禁言")
				data := map[string]any{"mute": map[string]any{"muted_until": mutedUntil}}
				var dataPtr any = data
				return httpx.ApiResponse[any]{Code: httpx.CodeConflict, Message: "您已被禁言", Data: &dataPtr}
			}
			observability.LogServerError(r, "Send", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = msg
		return httpx.ApiResponse[any]{Code: 0, Data: &data}
	})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	var req GetGroupMessagesHistoryRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "History", "请求体格式错误")
		return
	}
	if req.GroupID == "" || !isValidPagination(req.Pagination) {
		httpx.WriteInvalidRequest(w, r, "History", "参数错误")
		return
	}
	history, err := h.service.History(r.Context(), req.GroupID, req.Pagination)
	if err != nil {
		if errors.Is(err, ErrNotGroupMember) {
			httpx.WriteBusinessError(w, r, "History", httpx.CodeConflict, "不是群组成员")
			return
		}
		httpx.WriteAPIError(w, r, "History", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Update", "请求体格式错误")
		return
	}
	if req.MessageID == "" || req.Content == "" {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			observability.LogBusinessError(r, "Update", int(httpx.CodeNotFound), "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			observability.LogBusinessError(r, "Update", int(httpx.CodeForbidden), "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			observability.LogBusinessError(r, "Update", int(httpx.CodeConflict), "消息已被撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "消息已被撤回"))
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
	var req RecallGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Recall", "请求体格式错误")
		return
	}
	if req.MessageID == "" {
		httpx.WriteInvalidRequest(w, r, "Recall", "参数错误")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeNotFound), "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeForbidden), "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeForbidden, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeConflict), "消息已被撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "消息已被撤回"))
		case errors.Is(err, ErrMessageRecallWindow):
			observability.LogBusinessError(r, "Recall", int(httpx.CodeConflict), "撤回时间窗口已过")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeConflict, "撤回时间窗口已过"))
		default:
			observability.LogServerError(r, "Recall", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

// MarkRead is the unified mark_read endpoint for group messages. It replaces
// the legacy /me/mark_as_read and /me/mark_all_read endpoints. scope selects
// the operation: "all" clears every unread group message for the caller;
// "group" (the default) marks messages in group_id as read (up to the
// optional up_to_message_id).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkGroupReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "请求体格式错误")
		return
	}
	scope := req.Scope
	if scope == "" {
		scope = "group"
	}
	switch scope {
	case "all":
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	case "group":
		if req.GroupID == "" {
			httpx.WriteInvalidRequest(w, r, "MarkRead", "参数错误")
			return
		}
		if err := h.service.MarkAsRead(r.Context(), MarkGroupMessagesReadRequest{
			GroupID:       req.GroupID,
			UpToMessageID: req.UpToMessageID,
		}); err != nil {
			switch {
			case errors.Is(err, ErrMessageNotFound):
				observability.LogBusinessError(r, "MarkRead", int(httpx.CodeNotFound), "消息不存在")
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "消息不存在"))
			default:
				observability.LogServerError(r, "MarkRead", err)
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			}
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		observability.LogBusinessError(r, "MarkRead", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
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
	_, _, err := decodeGroupMessageCursor(p.Cursor)
	return err == nil
}
