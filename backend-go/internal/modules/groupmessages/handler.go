package groupmessages

import (
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"net/http"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req SendGroupMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" || req.Type == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "group_messages/send", func() httpx.ApiResponse[MessageSent] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			return httpx.ApiResponse[MessageSent]{Code: 10600, Message: "服务暂时不可用"}
		}
		return httpx.Success(msg)
	})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	var req GetGroupMessagesHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	history, err := h.service.History(r.Context(), req.GroupID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == "" || req.Content == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息已被撤回"))
		case errors.Is(err, ErrMessageNotEditable):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息不可编辑"))
		default:
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	var req RecallGroupMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MessageID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		case errors.Is(err, ErrPermissionDenied):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10200, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "消息已被撤回"))
		case errors.Is(err, ErrMessageRecallWindow):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10400, "撤回时间窗口已过"))
		default:
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	var req MarkGroupMessagesReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.MarkAsRead(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "消息不存在"))
		default:
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil && r.ContentLength != 0 {
		var discard struct{}
		if err := json.NewDecoder(r.Body).Decode(&discard); err != nil {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
	}
	if err := h.service.MarkAllRead(r.Context()); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
