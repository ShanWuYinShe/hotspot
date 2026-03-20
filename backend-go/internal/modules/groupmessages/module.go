package groupmessages

import (
	"github.com/go-chi/chi/v5"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct{ handler *Handler }

func NewModule(factory *persistence.Factory) *Module {
	return &Module{handler: NewHandler(NewServiceWithRepo(NewPostgresRepository(factory, nil)))}
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/group_messages", func(gr chi.Router) {
		gr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/send", m.handler.Send)
			group.Post("/get_history", m.handler.History)
		})
	})
}
