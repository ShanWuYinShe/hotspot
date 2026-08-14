package search

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
	Service *Service
}

func NewModule(factory *persistence.Factory, source appconfig.Source) *Module {
	repo := NewPostgresRepository(factory)
	service := NewServiceWithSource(repo, source)
	return &Module{handler: NewHandler(service), Service: service}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/search", func(searchRouter chi.Router) {
		searchRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/suggest", m.handler.Suggest)
		})
	})
}

// MountPublic mounts the anonymous search/suggest on the /public/v1 node
// (spec api_spec_public.md), rate limited per IP.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/search", func(searchRouter chi.Router) {
		searchRouter.Post("/suggest", m.handler.Suggest)
	})
}
