package moments

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	momentsHandler  *Handler
	commentsHandler *comments.Handler
	Service         *Service
	Comments        *comments.Service
}

func NewModule(factory *persistence.Factory, actorProvider actor.Provider, appConfig appconfig.Source, imageResolver ImageResolver) *Module {
	momentRepo := NewPersistenceRepository(factory)
	notificationService := notify.NewServiceWithRepo(notify.NewPersistenceRepository(factory))
	txRunner := persistence.NewTxRunner(factory)
	momentService := NewServiceWithDepsAndImagesAndConfig(momentRepo, notificationService, actorProvider, imageResolver, appConfig, txRunner)
	commentService := comments.NewServiceWithDeps(comments.NewPersistenceRepository(factory), momentService, notificationService, actorProvider, appConfig, txRunner)
	return &Module{
		momentsHandler:  NewHandler(momentService),
		commentsHandler: comments.NewHandler(commentService),
		Service:         momentService,
		Comments:        commentService,
	}
}

// CommentsService exposes the comments service for cross-module wiring (e.g.
// reactions target resolution). It returns the same instance used by the
// moments module's own /api/v1/comments routes.
func (m *Module) CommentsService() *comments.Service { return m.Comments }

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/moments", func(momentsRouter chi.Router) {
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/get_detail", m.momentsHandler.OpenTheDoor)
			group.Post("/search", m.momentsHandler.Search)
		})
		momentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.momentsHandler.Create)
			group.Post("/update", m.momentsHandler.Update)
			group.Post("/delete", m.momentsHandler.Delete)
			group.Post("/list_my", m.momentsHandler.ListMine)
			group.Post("/favorite", m.momentsHandler.Favorite)
			group.Post("/list_favorites", m.momentsHandler.ListFavorites)
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
			group.Post("/get_detail", m.commentsHandler.GetDetail)
		})
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.commentsHandler.Create)
			group.Post("/delete", m.commentsHandler.Delete)
		})
	})
}
