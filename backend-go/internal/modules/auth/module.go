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
	service *Service
}

func NewModule(ctx context.Context, cfg config.Config, factory *persistence.Factory, source appconfig.Source) *Module {
	authService := platformauth.NewServiceWithPersistence(cfg, factory)
	platformauth.SetDefaultService(authService)
	// Wire the configured bcrypt cost once at startup. config.Validate() has
	// already enforced the OWASP minimum by this point.
	ConfigureHashingCost(cfg.BcryptCost)
	repo := NewPostgresRepository(factory)
	service := NewServiceWithSource(ctx, authService, repo, cfg, source)
	return &Module{handler: NewHandler(service), service: service}
}

// SetLoginTracker 替换登录失败追踪器（P2-2）：Redis 可用时由 bootstrap 注入
// Redis 实现，保证多实例共享计数与重启不丢锁定状态；缺省为进程内实现。
func (m *Module) SetLoginTracker(tracker FailedLoginTracker) { m.service.loginTracker = tracker }

// Service 暴露模块服务（bootstrap 跨模块注入用，如 users.Upgrade 的
// 凭证写入，spec api_spec_user.md → upgrade）。
func (m *Module) Service() *Service { return m.service }

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/auth", func(authRouter chi.Router) {
		authRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountAuthenticatedRoutes(group, m.handler)
		})
	})
}

// MountPublic mounts the anonymous auth endpoints on the /public/v1 node
// (spec api_spec_public.md): no auth middleware, rate limited per IP.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/auth", func(authRouter chi.Router) {
		MountPublicRoutes(authRouter, m.handler)
	})
}
