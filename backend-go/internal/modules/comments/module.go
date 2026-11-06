package comments

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

// Module 是一等模块：评论路由/服务由本包显式装配与挂载，不再由 moments
// 模块内部构造（架构审计 P1-6：模块边界虚化）。依赖 moments 的 MomentProvider
// 由 bootstrap 在 moments 模块之后注入。
type Module struct {
	Handler *Handler
	Service *Service
}

// NewModule 组装评论模块（Postgres 仓储 + 统一事务运行器）并挂载路由用 handler。
func NewModule(factory *persistence.Factory, moments MomentProvider, notifications *notify.Service, actorProvider actor.Provider, appConfig appconfig.Source) *Module {
	service := NewServiceWithDeps(NewPostgresRepository(factory), moments, notifications, actorProvider, appConfig, persistence.NewTxRunner(factory))
	return &Module{Handler: NewHandler(service), Service: service}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/comments", func(commentsRouter chi.Router) {
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/fetch", m.Handler.List)
			group.Post("/fetch_replies", m.Handler.ListReplies)
			group.Post("/open_the_door", m.Handler.GetDetail)
			group.Post("/create", m.Handler.Create)
			group.Post("/delete", m.Handler.Delete)
		})
	})
}

// MountPublic mounts anonymous comment reads under /public/v1.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/comments", func(commentsRouter chi.Router) {
		commentsRouter.Post("/fetch", m.Handler.List)
		commentsRouter.Post("/fetch_replies", m.Handler.ListReplies)
		commentsRouter.Post("/open_the_door", m.Handler.GetDetail)
	})
}
