package share

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareType == "" || req.Payload == nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "share/create", func() httpx.ApiResponse[ShareCreated] {
		return httpx.Success(h.service.Create(r.Context(), req))
	})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var req GetShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareCode == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	detail, ok := h.service.Get(r.Context(), req.ShareCode)
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "分享不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(detail))
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.List(r.Context())))
}
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ShareID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if !h.service.Revoke(r.Context(), req.ShareID) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "分享不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}
