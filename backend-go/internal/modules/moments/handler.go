package moments

import (
	"encoding/json"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	var req CreateMomentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Description == "" || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "moments/create", func() httpx.ApiResponse[MomentItem] {
		return httpx.Success(h.service.Create(r.Context(), req))
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req MomentOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MomentID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok := h.service.OpenTheDoor(r.Context(), req.MomentID)
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "瞬间不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListMine(r.Context())))
}
