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
	repo := NewPostgresRepository(factory)
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
			group.Post("/following", m.handler.Following)
			group.Post("/nearby_moments", m.handler.Nearby)
			group.Post("/nearby_groups", m.handler.NearbyGroups)
			group.Post("/nearby_users", m.handler.NearbyUsers)
			group.Post("/trending", m.handler.Trending)
		})
	})
}

// MountPublic mounts the anonymous feed endpoints on the /public/v1 node
// (spec api_spec_public.md). The following feed needs a social graph, so it
// stays /api/v1-only.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/feed", func(feedRouter chi.Router) {
		feedRouter.Post("/nearby_groups", m.handler.NearbyGroups)
		feedRouter.Post("/nearby_users", m.handler.NearbyUsers)
		feedRouter.Post("/trending", m.handler.Trending)
	})
}
