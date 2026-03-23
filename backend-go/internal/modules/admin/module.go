package admin

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notifications"
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct{ handler *Handler }

func NewModule(factory *persistence.Factory, source appconfig.MutableSource) *Module {
	repo := NewPersistenceRepository(factory)
	provider := NewAppConfigProvider(source)
	systemRepo := systemmodule.NewPersistenceRepository(factory)
	moderationStore := NewPersistenceModerationStore(factory, systemRepo)
	pushService := notifications.NewServiceWithRepo(notifications.NewPersistenceRepository(factory))
	return NewModuleWithDeps(repo, provider, moderationStore, pushService)
}

func NewModuleWithDeps(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Module {
	service := NewService(repo, provider, moderationStore, pushService)
	return &Module{handler: NewHandler(service)}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/admin", func(adminRouter chi.Router) {
		adminRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountRoutes(group, m.handler)
		})
	})
}
