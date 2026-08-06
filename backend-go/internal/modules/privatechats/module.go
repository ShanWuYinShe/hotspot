package privatechats

import (
	"context"

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

// shareValidator 校验 share_code 有效性（由 share 模块实现，经 bootstrap 注入）。
type shareValidator interface {
	ValidateShareCode(ctx context.Context, shareCode string) error
}

type Module struct {
	chatHandler    *Handler
	messageHandler *privatemessages.Handler
	messageRuntime *privatemessages.Runtime
}

func NewModule(factory *persistence.Factory, source appconfig.Source, shares shareValidator) *Module {
	messageRuntime := privatemessages.NewRuntime(factory, source)
	messageRuntime.SetShareValidator(shares)
	userService := users.NewServiceWithRepo(users.NewPostgresRepository(factory))
	chatService := NewServiceWithDeps(NewPostgresRepository(factory), messageRuntime.ReadService(), userService)
	return &Module{chatHandler: NewHandler(chatService, source), messageHandler: messageRuntime.Handler(), messageRuntime: messageRuntime}
}

func (m *Module) MessageService() privatemessages.ReadService {
	return m.messageRuntime.ReadService()
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/private_chats", func(chatRouter chi.Router) {
		chatRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list", m.chatHandler.List)
			group.Post("/get_detail", m.chatHandler.OpenTheDoor)
			group.Post("/update_preferences", m.chatHandler.UpdateSettings)
			group.Post("/delete", m.chatHandler.Delete)
		})
	})
	r.Route("/api/v1/private_messages", func(messageRouter chi.Router) {
		messageRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/send", m.messageHandler.Send)
			group.Post("/get_history", m.messageHandler.GetHistory)
			group.Post("/update", m.messageHandler.Update)
			group.Post("/recall", m.messageHandler.Recall)
			group.Post("/mark_read", m.messageHandler.MarkRead)
		})
	})
}
