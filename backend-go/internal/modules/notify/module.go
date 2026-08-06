package notify

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
	repo := NewPostgresRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithRepo(repo))}
}

func NewModuleWithService(service *Service) *Module {
	return &Module{handler: NewHandler(service)}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/notify", func(notificationRouter chi.Router) {
		notificationRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list", m.handler.List)
			group.Post("/get_unread_count", m.handler.UnreadCount)
			group.Post("/mark_read", m.handler.MarkRead)
		})
	})
}
