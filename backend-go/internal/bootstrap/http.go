package bootstrap

import (
	"net/http"

	"github.com/go-chi/chi/v5"

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
	r.Use(cors.Middleware(cors.DefaultConfig()))
	r.Use(observability.RequestIDMiddleware)
	if logger != nil {
		r.Use(observability.RequestLoggerMiddleware(logger))
	}
	// Catch panics into a JSON envelope (never leak a stack trace) and cap
	// request bodies before they reach handlers.
	r.Use(httpx.Recoverer)
	r.Use(httpx.LimitRequestBody)
	r.NotFound(httpx.NotFoundHandler)
	r.MethodNotAllowed(httpx.MethodNotAllowedHandler)
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
