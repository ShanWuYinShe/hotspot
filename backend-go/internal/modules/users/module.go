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
	repo := NewPersistenceRepository(factory)
	notificationService := notify.NewServiceWithRepo(notify.NewPersistenceRepository(factory))
	svc := NewServiceWithDeps(repo, notificationService, appConfig)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/users", func(usersRouter chi.Router) {
		usersRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/search", m.handler.Search)
			group.Post("/get_detail", m.handler.OpenTheDoor)
			group.Post("/get_profile", m.handler.GetProfile)
			group.Post("/list_friends", m.handler.ListFriends)
			group.Post("/list_following", m.handler.ListFollowing)
			group.Post("/list_followers", m.handler.ListFollowers)
			group.Post("/list_groups", m.handler.ListGroups)
			group.Post("/list_moments", m.handler.ListMoments)
			group.Post("/list_reacted_moments", m.handler.ListReactedMoments)
			group.Post("/batch_get", m.handler.BatchGet)
		})
		usersRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/get_account", m.handler.GetMyProfile)
			group.Post("/update_account", m.handler.UpdateMyProfile)
			group.Post("/email/request_change", m.handler.RequestEmailChange)
			group.Post("/email/confirm_change", m.handler.ConfirmEmailChange)
			group.Post("/phone/request_change", m.handler.RequestPhoneChange)
			group.Post("/phone/confirm_change", m.handler.ConfirmPhoneChange)
			group.Post("/update_location", m.handler.UpdateLocation)
			group.Post("/upgrade", m.handler.Upgrade)
			group.Post("/get_settings", m.handler.GetMySettings)
			group.Post("/update_settings", m.handler.UpdateMySettings)
			group.Post("/delete_account", m.handler.DeleteAccount)
			group.Post("/follow", m.handler.Follow)
			group.Post("/block", m.handler.Block)
			group.Post("/list_blocked", m.handler.ListBlocked)
			group.Post("/set_remark", m.handler.SetRemark)
		})
	})
}
