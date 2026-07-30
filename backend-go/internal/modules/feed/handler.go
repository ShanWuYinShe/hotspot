package feed

import (
	"errors"
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

func (h *Handler) Following(w http.ResponseWriter, r *http.Request) {
	var req GetFeedFollowingRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.Following(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "Following", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) Nearby(w http.ResponseWriter, r *http.Request) {
	var req GetFeedNearbyRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.Nearby(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "Nearby", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) NearbyGroups(w http.ResponseWriter, r *http.Request) {
	var req GetFeedNearbyGroupsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.NearbyGroups(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "NearbyGroups", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) NearbyUsers(w http.ResponseWriter, r *http.Request) {
	var req GetFeedNearbyUsersRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.NearbyUsers(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "NearbyUsers", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) Trending(w http.ResponseWriter, r *http.Request) {
	var req struct{}
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	bundle, err := h.service.Trending(r.Context())
	if err != nil {
		handleFeedError(w, r, "Trending", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bundle))
}

func handleFeedError(w http.ResponseWriter, r *http.Request, method string, err error) {
	if errors.Is(err, ErrInvalidFeedRequest) || errors.Is(err, ErrInvalidCursor) {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	observability.L().Error("handler error",
		"method", method,
		"error", err,
		"request_id", observability.RequestIDFromContext(r.Context()),
	)
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
}
