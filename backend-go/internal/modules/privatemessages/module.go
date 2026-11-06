package privatemessages

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

// Module 是一等模块：私聊消息的路由/服务由本包显式装配与挂载，不再由
// privatechats 模块内部构造（架构审计 P1-6：模块边界虚化）。users 依赖以
// userReader 最小接口注入，避免模块间直接依赖具体 Service 类型。
type Module struct {
	Handler *Handler
	runtime *Runtime
}

// NewModule 组装私聊消息模块；shares 校验器与 users 读取器由 bootstrap 注入。
func NewModule(factory *persistence.Factory, source appconfig.Source, shares shareValidator, userService userReader) *Module {
	runtime := NewRuntime(factory, source, userService)
	runtime.SetShareValidator(shares)
	return &Module{Handler: runtime.Handler(), runtime: runtime}
}

// MessageService 暴露消息读服务给 privatechats（会话列表/打开会话）与
// reactions/reports 的跨模块读路径使用。
func (m *Module) MessageService() ReadService { return m.runtime.ReadService() }

// SetMessageReactionsSource 接线消息 reactions 聚合源。
func (m *Module) SetMessageReactionsSource(src MessageReactionsSource) {
	m.runtime.SetReactionsSource(src)
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/private_messages", func(messageRouter chi.Router) {
		messageRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/send", m.Handler.Send)
			group.Post("/fetch", m.Handler.List)
			group.Post("/update", m.Handler.Update)
			group.Post("/recall", m.Handler.Recall)
			group.Post("/mark_read", m.Handler.MarkRead)
		})
	})
}
