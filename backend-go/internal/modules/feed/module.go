package feed

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

func NewModule(factory *persistence.Factory, source appconfig.Source, imageResolver ImageResolver) *Module {
	repo := NewPersistenceRepository(factory)
	svc := NewServiceWithSourceAndImages(repo, source, imageResolver)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/feed", func(feedRouter chi.Router) {
		feedRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list_moments", m.handler.ListMoments)
			group.Post("/list_groups", m.handler.ListGroups)
			group.Post("/list_users", m.handler.ListUsers)
			group.Post("/dismiss", m.handler.Dismiss)
		})
		feedRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list_trending", m.handler.ListTrending)
		})
	})
}
