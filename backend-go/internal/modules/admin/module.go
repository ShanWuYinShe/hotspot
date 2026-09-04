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

func NewModule(factory *persistence.Factory, source appconfig.MutableSource, pushService *notify.Service) *Module {
	repo := NewPostgresRepository(factory)
	provider := NewAppConfigProvider(source)
	systemRepo := systemmodule.NewPostgresRepository(factory)
	moderationStore := NewPersistenceModerationStore(factory, systemRepo)
	return NewModuleWithDeps(repo, provider, moderationStore, pushService)
}

func NewModuleWithDeps(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Module {
	service := NewService(repo, provider, moderationStore, pushService)
	return &Module{handler: NewHandler(service), Repo: repo}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/admin/v1", func(adminRouter chi.Router) {
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

// tagBlockChecker 实现 groups 注入接口（IsBlocked）：新内容 tags 含被屏蔽标签 → 10400
// （spec api_spec_admin.md tags/set_ban「新内容约束」）。屏蔽（admin_blocked_tags 有行）即拦截。
type tagBlockChecker struct{ m *Module }

func (c tagBlockChecker) IsBlocked(tag string) bool {
	if c.m == nil || c.m.handler == nil || c.m.handler.service == nil {
		return false
	}
	blocked, _, err := c.m.handler.service.moderationStore.GetTagDeleteState(context.Background(), tag)
	return err == nil && blocked
}

// BlockedTagChecker 供 groups 模块注入（bootstrap 接线）。
func (m *Module) BlockedTagChecker() tagBlockChecker {
	return tagBlockChecker{m: m}
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
