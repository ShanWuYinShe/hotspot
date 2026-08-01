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
		logBusinessError(r, "Set", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if !ValidReactTarget(req.TargetType) {
		logBusinessError(r, "Set", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if err := h.service.Set(r.Context(), req); err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			logBusinessError(r, "Set", 10300, "反应目标不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "反应目标不存在"))
			return
		}
		if errors.Is(err, ErrInvalidEmoji) {
			logBusinessError(r, "Set", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "Set", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(struct{}{}))
}

// List handles POST /api/v1/reactions/list.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var req ListReactionsRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil || req.TargetID == "" {
		logBusinessError(r, "List", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	if !ValidReactTarget(req.TargetType) {
		logBusinessError(r, "List", 10000, "参数错误")
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
		return
	}
	result, err := h.service.List(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrReactionTargetNotFound) {
			logBusinessError(r, "List", 10300, "反应目标不存在")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10300, "反应目标不存在"))
			return
		}
		if errors.Is(err, ErrInvalidCursor) {
			logBusinessError(r, "List", 10000, "参数错误")
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(10000, "参数错误"))
			return
		}
		logServerError(r, "List", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(10600, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(result))
}

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code int, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}
