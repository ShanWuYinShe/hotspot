package system

import (
	"net/http"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
		observability.LogBusinessError(r, "Health", int(httpx.Code(code)), message)
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
		httpx.WriteAPIError(w, r, "ListAnnouncements", err)
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
	// Web 端设备标识下发（spec api_spec_base.md「Web 端 Cookie 契约」）：
	// bootstrap 是冷启动第一端点，未携带 device_id Cookie 时在此生成下发，
	// 保证后续 auth/register、auth/guest 从 Cookie 读取时已存在；已携带则沿用。
	platformauth.EnsureDeviceIDCookie(w, r)
	var req BootstrapRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "请求体格式错误"))
		return
	}
	bootstrapResolver := h.service.GetBootstrapResolver()
	bd := h.service.Bootstrap(r.Context(), bootstrapResolver)
	// 缓存契约（spec system.md → bootstrap）：客户端带 If-None-Match 且版本未变化时，
	// 返回精简响应（仅 bootstrap_version，meta/client_config/profile/unread_summary 省略，客户端用缓存）。
	if etag := r.Header.Get("If-None-Match"); etag != "" && etag == bd.BootstrapVersion {
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(BootstrapCacheHitData{BootstrapVersion: bd.BootstrapVersion}))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bd))
}
