package reactions

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/actor"
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

// NewModule wires the reactions Service with its persistence repo, the target
// providers (one per supported target kind), a notification writer and an
// actor provider. Pass nil for providers/notifications to disable side
// effects (e.g. in tests).
func NewModule(factory *persistence.Factory, providers []TargetProvider, notificationWriter reactionNotificationWriter, actorProvider actor.Provider) *Module {
	repo := NewPostgresRepository(factory)
	svc := NewServiceWithDeps(repo, providers, notificationWriter, actorProvider)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/reactions", func(reactionsRouter chi.Router) {
		reactionsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list", m.handler.List)
		})
		reactionsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/set", m.handler.Set)
		})
	})
}
