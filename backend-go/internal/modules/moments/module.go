package moments

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/comments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	momentsHandler  *Handler
	commentsHandler *comments.Handler
}

func NewModule(factory *persistence.Factory) *Module {
	momentRepo := NewPersistenceRepository(factory)
	momentService := NewServiceWithRepo(momentRepo)
	commentService := comments.NewService(comments.NewInMemoryRepository(), momentService)
	return &Module{
		momentsHandler:  NewHandler(momentService),
		commentsHandler: comments.NewHandler(commentService),
	}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/moments", func(momentsRouter chi.Router) {
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/open_the_door", m.momentsHandler.OpenTheDoor)
		})
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.momentsHandler.Create)
			group.Post("/me/list", m.momentsHandler.ListMine)
		})
	})

	r.Route("/api/v1/comments", func(commentsRouter chi.Router) {
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list", m.commentsHandler.List)
		})
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.commentsHandler.Create)
		})
	})
}
