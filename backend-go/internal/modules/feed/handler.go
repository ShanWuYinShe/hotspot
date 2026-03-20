package feed

import (
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListMoments(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListMoments(r.Context())))
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListGroups(r.Context())))
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListUsers(r.Context())))
}

func (h *Handler) ListTrending(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.ListTrending(r.Context())))
}
