package system

import (
	"net/http"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/buildinfo"
	"hotspot/backend-go/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetMeta(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteCodeError(w, r, err.Code, err.Message)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(h.service.GetMeta(r.Context())))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteCodeError(w, r, err.Code, err.Message)
		return
	}
	code, message, data := h.service.Health(r.Context())
	if code != 0 {
		// 已由 WriteCodeError 统一记录业务错误日志/指标。
		httpx.WriteCodeError(w, r, httpx.Code(code), message)
		return
	}
	// 版本指纹随 health 下发（spec api_spec_types.md HealthStatus）：线上排障
	// 时从公网即可确认当前运行产物，无需登机查容器。
	data.Version = buildinfo.Version
	data.Commit = buildinfo.Commit
	data.BuildTime = buildinfo.BuildTime
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(*data))
}

func (h *Handler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequireEmptyObject(r); err != nil {
		httpx.WriteCodeError(w, r, err.Code, err.Message)
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
		httpx.WriteCodeError(w, r, err.Code, err.Message)
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
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	bootstrapResolver := h.service.GetBootstrapResolver()
	bd := h.service.Bootstrap(r.Context(), bootstrapResolver)
	// 缓存契约（spec system.md → bootstrap）：客户端带 If-None-Match 且版本
	// 未变化时返回精简响应——仅省略公共段 meta/client_config（客户端用
	// 缓存）；profile/unread_summary 为用户私有数据、无缓存语义，命中时
	// 仍恒返回最新值（见 BootstrapCacheHitData 注释，审查 P1-2）。
	if etag := r.Header.Get("If-None-Match"); etag != "" && etag == bd.BootstrapVersion {
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(BootstrapCacheHitData{BootstrapVersion: bd.BootstrapVersion, Profile: bd.Profile, UnreadSummary: bd.UnreadSummary}))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(bd))
}

// PublicBootstrap 是 /public/v1/system/bootstrap 的匿名冷启动实现
// （spec api_spec_public.md → PublicBootstrapData）：只返回公共段；
// 即使可选鉴权带来了 principal，也绝不下发 profile / unread_summary。
func (h *Handler) PublicBootstrap(w http.ResponseWriter, r *http.Request) {
	platformauth.EnsureDeviceIDCookie(w, r)
	var req BootstrapRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteCodeError(w, r, httpx.CodeInvalidRequest, "请求体格式错误")
		return
	}
	bd := h.service.Bootstrap(r.Context(), h.service.GetBootstrapResolver())
	if etag := r.Header.Get("If-None-Match"); etag != "" && etag == bd.BootstrapVersion {
		httpx.WriteJSON(w, http.StatusOK, httpx.Success(PublicBootstrapCacheHitData{BootstrapVersion: bd.BootstrapVersion}))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(PublicBootstrapData{BootstrapVersion: bd.BootstrapVersion, Meta: bd.Meta, ClientConfig: bd.ClientConfig}))
}
