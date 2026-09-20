package bootstrap

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/cors"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/ratelimit"
)

func NewHTTPRouter(modules []modulekit.Module) http.Handler {
	return newHTTPRouter(modules, nil)
}

func NewHTTPRouterWithLogger(modules []modulekit.Module, logger *observability.Logger) http.Handler {
	return newHTTPRouter(modules, logger)
}

func newHTTPRouter(modules []modulekit.Module, logger *observability.Logger) http.Handler {
	r := chi.NewRouter()
	// CORS 白名单经 CORS_ALLOWED_ORIGINS 装配：同源部署默认为空（不响应跨源）。
	corsCfg := cors.DefaultConfig().WithAllowedOrigins(config.Load().CorsAllowedOrigins)
	r.Use(cors.Middleware(corsCfg))
	r.Use(observability.RequestIDMiddleware)
	if logger != nil {
		r.Use(observability.RequestLoggerMiddleware(logger))
	}
	// Catch panics into a JSON envelope (never leak a stack trace) and cap
	// request bodies before they reach handlers.
	r.Use(httpx.Recoverer)
	r.Use(httpx.LimitRequestBody)
	// 请求计数/耗时指标（chi 路由模板维度）。放在 Recoverer 之后：panic
	// 恢复为 500 的请求同样被计数。
	r.Use(observability.RequestMetricsMiddleware)
	r.NotFound(httpx.NotFoundHandler)
	r.MethodNotAllowed(httpx.MethodNotAllowedHandler)
	// Prometheus 抓取端点：仅 compose 网络内可达（nginx 不反代 /metrics，
	// 外部访问落到 SPA 兜底）；可选采集栈见 docker-compose.yml observability
	// profile。不挂限流——15s 抓取节奏不会触发限流桶，也不应被其干扰。
	r.Method(http.MethodGet, "/metrics", observability.MetricsHandler())
	for _, module := range modules {
		module.Mount(r)
	}
	// /public/v1 匿名节点（spec api_spec_public.md）：纯匿名、不解析 Token，
	// 限流按 IP 维度（ratelimit 内部按路径分类），无资源守卫。
	publicRouter := chi.NewRouter()
	publicRouter.Use(ratelimit.Middleware)
	publicRouter.Use(idempotency.Middleware)
	publicRouter.NotFound(httpx.NotFoundHandler)
	publicRouter.MethodNotAllowed(httpx.MethodNotAllowedHandler)
	for _, module := range modules {
		if pm, ok := module.(modulekit.PublicMounter); ok {
			pm.MountPublic(publicRouter)
		}
	}
	r.Mount("/public/v1", publicRouter)
	return r
}
