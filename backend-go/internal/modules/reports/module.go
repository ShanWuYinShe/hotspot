package reports

import (
	"github.com/go-chi/chi/v5"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct{ handler *Handler }

func NewModule() *Module {
	return &Module{handler: NewHandler(NewServiceWithRepo(NewInMemoryRepository()))}
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/reports", func(rr chi.Router) {
		rr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/me/list", m.handler.List)
		})
	})
}
