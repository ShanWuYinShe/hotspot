package reactions

import (
	"errors"
	"net/http"

	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Set handles POST /api/v1/reactions/set (two-state unified endpoint).
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	var req SetReactionRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetID == "" || req.Emoji == "" {
		httpx.WriteInvalidRequest(w, r, "Set", "参数错误")
		return
	}
	if !ValidReactTarget(req.TargetType) {
		httpx.WriteInvalidRequest(w, r, "Set", "参数错误")
		return
	}
	if err := h.service.Set(r.Context(), req); err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			httpx.WriteBusinessError(w, r, "Set", httpx.CodeNotFound, "反应目标不存在")
			return
		}
		if errors.Is(err, ErrInvalidEmoji) {
			httpx.WriteInvalidRequest(w, r, "Set", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "Set", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

// List handles POST /api/v1/reactions/fetch.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListReactionsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetID == "" {
		httpx.WriteInvalidRequest(w, r, "List", "参数错误")
		return
	}
	if !ValidReactTarget(req.TargetType) {
		httpx.WriteInvalidRequest(w, r, "List", "参数错误")
		return
	}
	result, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			httpx.WriteBusinessError(w, r, "List", httpx.CodeNotFound, "反应目标不存在")
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			httpx.WriteInvalidRequest(w, r, "List", "参数错误")
			return
		}
		httpx.WriteAPIError(w, r, "List", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
