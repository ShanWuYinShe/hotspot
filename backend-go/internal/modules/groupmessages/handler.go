package groupmessages

import (
	"encoding/json"
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
