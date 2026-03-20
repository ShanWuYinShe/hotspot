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
		return httpx.Success(h.service.Send(r.Context(), req))
	})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	var req GetGroupMessagesHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.History(r.Context(), req.GroupID)))
}
