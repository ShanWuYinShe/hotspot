package mapmodule

import (
	"github.com/go-chi/chi/v5"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory) *Module {
	repo := NewPersistenceRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithRepo(repo))}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/map", func(mapRouter chi.Router) {
		mapRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/canvas", m.handler.Canvas)
		})
	})
}
