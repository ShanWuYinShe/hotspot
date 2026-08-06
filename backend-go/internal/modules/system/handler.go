package system

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

func logServerError(r *http.Request, method string, err error) {
	observability.L().Error("handler error", "method", method, "error", err, "request_id", observability.RequestIDFromContext(r.Context()))
}

func logBusinessError(r *http.Request, method string, code httpx.Code, message string) {
	observability.L().Warn("handler business error", "method", method, "code", code, "message", message, "request_id", observability.RequestIDFromContext(r.Context()))
}

func (h *Handler) GetMeta(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.GetMeta(r.Context())))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	code, message, data := h.service.Health(r.Context())
	if code != 0 {
		logBusinessError(r, "Health", httpx.Code(code), message)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.Code(code), message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(*data))
}

func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	payload, err := h.service.ListAnnouncements(r.Context())
	if err != nil {
		logServerError(r, "ListAnnouncements", err)
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(payload))
}

func (h *Handler) GetClientConfig(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(err.Code, err.Message))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.GetClientConfig(r.Context())))
}

func (h *Handler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var req BootstrapRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	bootstrapResolver := h.service.GetBootstrapResolver()
	bd := h.service.Bootstrap(r.Context(), bootstrapResolver)
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bd))
}
