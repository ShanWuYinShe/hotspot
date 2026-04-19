package privatechats

import (
	"github.com/go-chi/chi/v5"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	chatHandler    *Handler
	messageHandler *private_messages.Handler
}

func NewModule(factory *persistence.Factory, source appconfig.Source) *Module {
	messageRuntime := private_messages.NewRuntime(factory)
	userService := users.NewServiceWithRepo(users.NewPersistenceRepository(factory))
	chatService := NewServiceWithDeps(NewPostgresRepository(factory), messageRuntime.ReadService(), userService)
	return &Module{chatHandler: NewHandler(chatService, source), messageHandler: messageRuntime.Handler()}
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
			group.Post("/me/update_settings", m.chatHandler.UpdateSettings)
			group.Post("/me/delete", m.chatHandler.Delete)
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
			group.Post("/me/mark_as_read", m.messageHandler.MarkAsRead)
			group.Post("/me/mark_all_read", m.messageHandler.MarkAllRead)
		})
	})
}
