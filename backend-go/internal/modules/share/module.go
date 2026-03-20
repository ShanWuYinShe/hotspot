package share

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
	r.Route("/api/v1/share", func(sr chi.Router) {
		sr.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/get", m.handler.Get)
		})
		sr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/me/list", m.handler.List)
			group.Post("/revoke", m.handler.Revoke)
		})
	})
}
