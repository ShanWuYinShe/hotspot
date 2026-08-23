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
	momentRepo := NewPostgresRepository(factory)
	notificationService := notify.NewServiceWithRepo(notify.NewPostgresRepository(factory))
	txRunner := persistence.NewTxRunner(factory)
	momentService := NewServiceWithDepsAndImagesAndConfig(momentRepo, notificationService, actorProvider, imageResolver, appConfig, txRunner)
	commentService := comments.NewServiceWithDeps(comments.NewPostgresRepository(factory), momentService, notificationService, actorProvider, appConfig, txRunner)
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
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/open_the_door", m.momentsHandler.OpenTheDoor)
			group.Post("/search", m.momentsHandler.Search)
			group.Post("/create", m.momentsHandler.Create)
			group.Post("/update", m.momentsHandler.Update)
			group.Post("/delete", m.momentsHandler.Delete)
			group.Post("/fetch_created", m.momentsHandler.ListCreated)
			group.Post("/fetch_following", m.momentsHandler.FetchFollowing)
			group.Post("/fetch_nearby", m.momentsHandler.FetchNearby)
			group.Post("/set_favorite", m.momentsHandler.Favorite)
			group.Post("/fetch_favorites", m.momentsHandler.ListFavorites)
		})
	})

	r.Route("/api/v1/comments", func(commentsRouter chi.Router) {
		commentsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/fetch", m.commentsHandler.List)
			group.Post("/fetch_replies", m.commentsHandler.ListReplies)
			group.Post("/open_the_door", m.commentsHandler.GetDetail)
			group.Post("/create", m.commentsHandler.Create)
			group.Post("/delete", m.commentsHandler.Delete)
		})
	})
}

// MountPublic mounts the anonymous moments/comments endpoints on the
// /public/v1 node (spec api_spec_public.md).
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/moments", func(momentsRouter chi.Router) {
		momentsRouter.Post("/open_the_door", m.momentsHandler.OpenTheDoor)
		momentsRouter.Post("/search", m.momentsHandler.Search)
		// 附近公开瞬间的可见性不依赖身份（仅 privacy_level='public'），
		// 匿名版为拉新路径（spec api_spec_public.md）。
		momentsRouter.Post("/fetch_nearby", m.momentsHandler.FetchNearby)
	})
	r.Route("/comments", func(commentsRouter chi.Router) {
		commentsRouter.Post("/fetch", m.commentsHandler.List)
		commentsRouter.Post("/fetch_replies", m.commentsHandler.ListReplies)
		commentsRouter.Post("/open_the_door", m.commentsHandler.GetDetail)
	})
}
