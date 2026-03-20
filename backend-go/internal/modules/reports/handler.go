package reports

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
	var req CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetType == "" || req.TargetID == "" || req.Reason == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "reports/create", func() httpx.ApiResponse[ReportCreated] {
		return httpx.Success(h.service.Create(r.Context(), req))
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.List(r.Context())))
}
