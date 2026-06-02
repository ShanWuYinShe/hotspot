package bootstrap

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/cors"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
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
	for _, module := range modules {
		module.Mount(r)
	}
	return r
}
