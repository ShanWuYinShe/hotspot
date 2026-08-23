package reports

import (
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	handler *Handler
	Service *Service
}

func NewModule(factory *persistence.Factory) *Module {
	repo := NewPostgresRepository(factory)
	service := NewServiceWithRepo(repo)
	return &Module{handler: NewHandler(service), Service: service}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/reports", func(rr chi.Router) {
		rr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/fetch", m.handler.List)
		})
	})
}
