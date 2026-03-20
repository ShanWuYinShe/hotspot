package groups

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
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Location.LocationName == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	idempotency.ReplayOrStore(w, r, "groups/create", func() httpx.ApiResponse[GroupItem] {
		return httpx.Success(h.service.Create(r.Context(), req))
	})
}

func (h *Handler) OpenTheDoor(w http.ResponseWriter, r *http.Request) {
	var req GroupOpenTheDoorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GroupID == "" {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	item, ok := h.service.OpenTheDoor(r.Context(), req.GroupID)
	if !ok {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "群组不存在"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(item))
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchGroupsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "请求体格式错误"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.Search(r.Context(), req.Keyword)))
}
