package feed

import (
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListMoments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMoments(r.Context())
	if err != nil {
		observability.L().Error("handler error",
			"method", "ListMoments",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListGroups(r.Context())
	if err != nil {
		observability.L().Error("handler error",
			"method", "ListGroups",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListUsers(r.Context())
	if err != nil {
		observability.L().Error("handler error",
			"method", "ListUsers",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(items))
}

func (h *Handler) ListTrending(w http.ResponseWriter, r *http.Request) {
	bundle, err := h.service.ListTrending(r.Context())
	if err != nil {
		observability.L().Error("handler error",
			"method", "ListTrending",
			"error", err,
			"request_id", observability.RequestIDFromContext(r.Context()),
		)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bundle))
}
