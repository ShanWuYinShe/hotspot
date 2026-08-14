package users

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
	Service *Service
}

func NewModule(factory *persistence.Factory, appConfig appconfig.Source) *Module {
	repo := NewPostgresRepository(factory)
	notificationService := notify.NewServiceWithRepo(notify.NewPostgresRepository(factory))
	svc := NewServiceWithDeps(repo, notificationService, appConfig)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/users", func(usersRouter chi.Router) {
		usersRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/search", m.handler.Search)
			group.Post("/open_the_door", m.handler.OpenTheDoor)
			group.Post("/fetch_friends", m.handler.ListFriends)
			group.Post("/fetch_following", m.handler.ListFollowing)
			group.Post("/fetch_followers", m.handler.ListFollowers)
			group.Post("/fetch_groups", m.handler.ListGroups)
			group.Post("/fetch_moments", m.handler.ListMoments)
			group.Post("/fetch_reacted_moments", m.handler.ListReactedMoments)
			group.Post("/batch_fetch", m.handler.BatchGet)
			group.Post("/account", m.handler.GetMyProfile)
			group.Post("/update_account", m.handler.UpdateMyProfile)
			group.Post("/update_location", m.handler.UpdateLocation)
			group.Post("/upgrade", m.handler.Upgrade)
			group.Post("/settings", m.handler.GetMySettings)
			group.Post("/update_settings", m.handler.UpdateMySettings)
			group.Post("/delete_account", m.handler.DeleteAccount)
			group.Post("/set_follow", m.handler.Follow)
			group.Post("/set_block", m.handler.Block)
			group.Post("/fetch_blocked", m.handler.ListBlocked)
			group.Post("/set_remark", m.handler.SetRemark)
		})
	})
}

// MountPublic mounts the anonymous users endpoints on the /public/v1 node
// (spec api_spec_public.md). Same handlers, anonymous (viewer-less) view.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/users", func(usersRouter chi.Router) {
		usersRouter.Post("/search", m.handler.Search)
		usersRouter.Post("/open_the_door", m.handler.OpenTheDoor)
		usersRouter.Post("/fetch_friends", m.handler.ListFriends)
		usersRouter.Post("/fetch_following", m.handler.ListFollowing)
		usersRouter.Post("/fetch_followers", m.handler.ListFollowers)
		usersRouter.Post("/fetch_groups", m.handler.ListGroups)
		usersRouter.Post("/fetch_moments", m.handler.ListMoments)
		usersRouter.Post("/fetch_reacted_moments", m.handler.ListReactedMoments)
		usersRouter.Post("/batch_fetch", m.handler.BatchGet)
	})
}
