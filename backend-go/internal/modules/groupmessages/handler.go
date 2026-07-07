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
	var req SendGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Send", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.GroupID == "" || req.Type == "" {
		logBusinessError(r, "Send", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.SenderUserID = principal.UserID
	idempotency.ReplayOrStore(w, r, "group_messages/send", func() httpx.ApiResponse[MessageSent] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrNotGroupMember) {
				logBusinessError(r, "Send", 10400, "不是群组成员")
				return httpx.ApiResponse[MessageSent]{Code: 10400, Message: "不是群组成员"}
			}
			if errors.Is(err, ErrUserMuted) {
				mutedUntil := ""
				var mutedErr *MutedError
				if errors.As(err, &mutedErr) {
					mutedUntil = mutedErr.MutedUntil
				}
				message := "您已被禁言"
				if mutedUntil != "" {
					message = "您已被禁言，截至时间：" + mutedUntil
				}
				logBusinessError(r, "Send", 10400, message)
				return httpx.ApiResponse[MessageSent]{Code: 10400, Message: message}
			}
			logServerError(r, "Send", err)
			return httpx.ApiResponse[MessageSent]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(msg)
	})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	var req GetGroupMessagesHistoryRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "History", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.GroupID == "" || !isValidPagination(req.Pagination) {
		logBusinessError(r, "History", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	history, err := h.service.History(r.Context(), req.GroupID, req.Pagination)
	if err != nil {
		if errors.Is(err, ErrNotGroupMember) {
			logBusinessError(r, "History", 10400, "不是群组成员")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "不是群组成员"))
			return
		}
		logServerError(r, "History", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Update", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.MessageID == "" || req.Content == "" {
		logBusinessError(r, "Update", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			logBusinessError(r, "Update", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			logBusinessError(r, "Update", 10200, "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			logBusinessError(r, "Update", 10400, "消息已被撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息已被撤回"))
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
	var req RecallGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "Recall", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.MessageID == "" {
		logBusinessError(r, "Recall", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			logBusinessError(r, "Recall", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			logBusinessError(r, "Recall", 10200, "权限不足")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			logBusinessError(r, "Recall", 10400, "消息已被撤回")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息已被撤回"))
		case errors.Is(err, ErrMessageRecallWindow):
			logBusinessError(r, "Recall", 10400, "撤回时间窗口已过")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "撤回时间窗口已过"))
		default:
			logServerError(r, "Recall", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	var req MarkGroupMessagesReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		logBusinessError(r, "MarkAsRead", 10000, "请求体格式错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	if req.GroupID == "" {
		logBusinessError(r, "MarkAsRead", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.MarkAsRead(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			logBusinessError(r, "MarkAsRead", 10300, "消息不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		default:
			logServerError(r, "MarkAsRead", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
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
