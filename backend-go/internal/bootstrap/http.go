package bootstrap

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
)

func NewHTTPRouter(modules []modulekit.Module) http.Handler {
	r := chi.NewRouter()
	r.Use(observability.RequestIDMiddleware)
	for _, module := range modules {
		module.Mount(r)
	}
	return r
}
