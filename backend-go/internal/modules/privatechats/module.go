package privatechats

import (
	"github.com/go-chi/chi/v5"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	chatHandler    *Handler
	messageHandler *private_messages.Handler
}

func NewModule(cfg config.Config, factory *persistence.Factory) *Module {
	appCfg := appconfig.NewStaticSource(cfg)
	messageRepo := private_messages.NewPostgresRepository(factory, nil)
	messageService := private_messages.NewServiceWithRepo(messageRepo)
	chatService := NewServiceWithRepos(NewPostgresRepository(factory, nil), messageService)
	return &Module{chatHandler: NewHandler(chatService, appCfg), messageHandler: private_messages.NewHandler(messageService)}
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/private_chats", func(chatRouter chi.Router) {
		chatRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/me/list", m.chatHandler.List)
			group.Post("/open_the_door", m.chatHandler.OpenTheDoor)
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
		})
	})
}
