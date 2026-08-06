package auth

import (
	"context"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(ctx context.Context, cfg config.Config, factory *persistence.Factory, source appconfig.Source) *Module {
	authService := platformauth.NewServiceWithPersistence(cfg, factory)
	platformauth.SetDefaultService(authService)
	// Wire the configured bcrypt cost once at startup. config.Validate() has
	// already enforced the OWASP minimum by this point.
	ConfigureHashingCost(cfg.BcryptCost)
	repo := NewPostgresRepository(factory)
	service := NewServiceWithSource(ctx, authService, repo, cfg, source)
	return &Module{handler: NewHandler(service)}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/auth", func(authRouter chi.Router) {
		authRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountRoutes(group, m.handler)
		})

		authRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountAuthenticatedRoutes(group, m.handler)
		})
	})
}
