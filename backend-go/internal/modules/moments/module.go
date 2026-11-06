package moments

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

type Module struct {
	momentsHandler *Handler
	Service        *Service
}

func NewModule(factory *persistence.Factory, actorProvider actor.Provider, appConfig appconfig.Source, imageResolver ImageResolver, notificationService *notify.Service) *Module {
	momentRepo := NewPostgresRepository(factory)
	txRunner := persistence.NewTxRunner(factory)
	momentService := NewServiceWithDepsAndImagesAndConfig(momentRepo, notificationService, actorProvider, imageResolver, appConfig, txRunner)
	return &Module{momentsHandler: NewHandler(momentService), Service: momentService}
}

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

}

// MountPublic mounts the anonymous moments endpoints on the
// /public/v1 node (spec api_spec_public.md).
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/moments", func(momentsRouter chi.Router) {
		momentsRouter.Post("/open_the_door", m.momentsHandler.PublicOpenTheDoor)
		momentsRouter.Post("/search", m.momentsHandler.PublicSearch)
		// 附近公开瞬间的可见性不依赖身份（仅 privacy_level='public'），
		// 匿名版为拉新路径（spec api_spec_public.md）。
		momentsRouter.Post("/fetch_nearby", m.momentsHandler.PublicFetchNearby)
	})
}
