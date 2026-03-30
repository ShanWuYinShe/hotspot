package feed

import (
	"encoding/json"
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
	var req ListFeedMomentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.ListMoments(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "ListMoments", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	var req ListFeedGroupsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.ListGroups(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "ListGroups", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var req ListFeedUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	resp, err := h.service.ListUsers(r.Context(), req)
	if err != nil {
		handleFeedError(w, r, "ListUsers", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(resp))
}

func (h *Handler) ListTrending(w http.ResponseWriter, r *http.Request) {
	var req struct{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	bundle, err := h.service.ListTrending(r.Context())
	if err != nil {
		handleFeedError(w, r, "ListTrending", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bundle))
}

func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	var req DismissFeedItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Dismiss(r.Context(), req); err != nil {
		switch err {
		case ErrInvalidFeedRequest, ErrInvalidCursor:
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		case ErrDismissTargetNotFound:
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "资源不存在"))
		default:
			observability.L().Error("handler error",
				"method", "Dismiss",
				"error", err,
				"request_id", observability.RequestIDFromContext(r.Context()),
			)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func handleFeedError(w http.ResponseWriter, r *http.Request, method string, err error) {
	if err == ErrInvalidFeedRequest || err == ErrInvalidCursor {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err == ErrDismissTargetNotFound {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "资源不存在"))
		return
	}
	observability.L().Error("handler error",
		"method", method,
		"error", err,
		"request_id", observability.RequestIDFromContext(r.Context()),
	)
	httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
}
