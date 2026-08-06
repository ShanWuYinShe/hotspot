package reactions

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

// Set handles POST /api/v1/reactions/set (two-state unified endpoint).
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	var req SetReactionRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetID == "" || req.Emoji == "" {
		observability.LogBusinessError(r, "Set", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if !ValidReactTarget(req.TargetType) {
		observability.LogBusinessError(r, "Set", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if err := h.service.Set(r.Context(), req); err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			observability.LogBusinessError(r, "Set", int(httpx.CodeNotFound), "反应目标不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "反应目标不存在"))
			return
		}
		if errors.Is(err, ErrInvalidEmoji) {
			observability.LogBusinessError(r, "Set", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		observability.LogServerError(r, "Set", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

// List handles POST /api/v1/reactions/list.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListReactionsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetID == "" {
		observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	if !ValidReactTarget(req.TargetType) {
		observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return
	}
	result, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			observability.LogBusinessError(r, "List", int(httpx.CodeNotFound), "反应目标不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, "反应目标不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			observability.LogBusinessError(r, "List", int(httpx.CodeInvalidRequest), "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
			return
		}
		observability.LogServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}
