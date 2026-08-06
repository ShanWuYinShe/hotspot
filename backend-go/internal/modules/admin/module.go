package admin

import (
	"context"

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
	// Service (BanChecker), enforcing httpx.CodeAccountBanned for banned accounts.
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

// AppendGroupDismissAudit implements groups.GroupDismissAuditWriter so the
// groups module can write owner-dismiss audit entries without a groups→admin
// import cycle (spec api_spec_group.md).
func (m *Module) AppendGroupDismissAudit(ctx context.Context, operatorID string, groupID string, groupName string, reason string) error {
	if m == nil || m.handler == nil || m.handler.service == nil {
		return nil
	}
	return m.handler.service.AppendGroupDismissAudit(ctx, operatorID, groupID, groupName, reason)
}
