package privatechats

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	chatHandler *Handler
	service     *Service
}

// NewModule 组装私聊会话模块。消息能力由 privatemessages 模块以 ReadService
// 最小接口注入（模块边界：会话模块不构造消息模块，也不再私建其运行时）。
func NewModule(factory *persistence.Factory, source appconfig.Source, userService *users.Service, messages privatemessages.ReadService) *Module {
	chatService := NewServiceWithDeps(NewPostgresRepository(factory), messages, userService)
	return &Module{chatHandler: NewHandler(chatService, source), service: chatService}
}

// Service 暴露会话服务（bootstrap 注入账户注销级联等跨模块能力）。
func (m *Module) Service() *Service { return m.service }

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/private_chats", func(chatRouter chi.Router) {
		chatRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/fetch", m.chatHandler.List)
			group.Post("/open_the_door", m.chatHandler.OpenTheDoor)
			group.Post("/update_preferences", m.chatHandler.UpdateSettings)
			group.Post("/delete", m.chatHandler.Delete)
		})
	})
}
