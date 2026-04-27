package moments

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notifications"
	"hotspot/backend-go/internal/platform/actor"
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

func NewModule(factory *persistence.Factory, actorProvider actor.Provider) *Module {
	momentRepo := NewPersistenceRepository(factory)
	notificationService := notifications.NewServiceWithRepo(notifications.NewPersistenceRepository(factory))
	momentService := NewServiceWithDeps(momentRepo, notificationService, actorProvider)
	commentService := comments.NewServiceWithDeps(comments.NewPersistenceRepository(factory), momentService, notificationService, actorProvider)
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
			group.Post("/search", m.momentsHandler.Search)
			group.Post("/list_likes", m.momentsHandler.ListLikes)
		})
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.momentsHandler.Create)
			group.Post("/update", m.momentsHandler.Update)
			group.Post("/delete", m.momentsHandler.Delete)
			group.Post("/me/list", m.momentsHandler.ListMine)
			group.Post("/me/like", m.momentsHandler.Like)
			group.Post("/me/unlike", m.momentsHandler.Unlike)
			group.Post("/me/favorite", m.momentsHandler.Favorite)
			group.Post("/me/unfavorite", m.momentsHandler.Unfavorite)
			group.Post("/me/list_favorites", m.momentsHandler.ListFavorites)
		})
	})

	r.Route("/api/v1/comments", func(commentsRouter chi.Router) {
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/list", m.commentsHandler.List)
			group.Post("/list_replies", m.commentsHandler.ListReplies)
			group.Post("/list_likes", m.commentsHandler.ListLikes)
		})
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.commentsHandler.Create)
			group.Post("/delete", m.commentsHandler.Delete)
			group.Post("/me/like", m.commentsHandler.Like)
			group.Post("/me/unlike", m.commentsHandler.Unlike)
		})
	})
}
