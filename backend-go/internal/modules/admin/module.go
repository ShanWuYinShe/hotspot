package admin

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notify"
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
	// Repo is exposed so bootstrap can wire its IsBanned into the platform auth
	// Service (BanChecker), enforcing 10201 for banned accounts.
	Repo Repository
}

func NewModule(factory *persistence.Factory, source appconfig.MutableSource) *Module {
	repo := NewPersistenceRepository(factory)
	provider := NewAppConfigProvider(source)
	systemRepo := systemmodule.NewPersistenceRepository(factory)
	moderationStore := NewPersistenceModerationStore(factory, systemRepo)
	pushService := notify.NewServiceWithRepo(notify.NewPersistenceRepository(factory))
	return NewModuleWithDeps(repo, provider, moderationStore, pushService)
}

func NewModuleWithDeps(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Module {
	service := NewService(repo, provider, moderationStore, pushService)
	return &Module{handler: NewHandler(service), Repo: repo}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/admin", func(adminRouter chi.Router) {
		adminRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			// AdminAccess is a middleware backstop on top of the service-layer RBAC:
			// it rejects any request whose principal is not an admin before it reaches
			// the handler. Belt and suspenders — the service layer also enforces this.
			group.Use(platformauth.AdminAccess)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountRoutes(group, m.handler)
		})
	})
}
