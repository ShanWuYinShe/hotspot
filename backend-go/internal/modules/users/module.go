package users

import (
	"github.com/go-chi/chi/v5"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory) *Module {
	repo := NewPersistenceRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithRepo(repo))}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/users", func(usersRouter chi.Router) {
		usersRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/search", m.handler.Search)
			group.Post("/open_the_door", m.handler.OpenTheDoor)
			group.Post("/list_friends", m.handler.ListFriends)
			group.Post("/list_following", m.handler.ListFollowing)
			group.Post("/list_followers", m.handler.ListFollowers)
			group.Post("/list_groups", m.handler.ListGroups)
			group.Post("/list_moments", m.handler.ListMoments)
			group.Post("/list_liked_moments", m.handler.ListLikedMoments)
		})
		usersRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/me/profile/get", m.handler.GetMyProfile)
			group.Post("/me/profile/update", m.handler.UpdateMyProfile)
			group.Post("/me/email/request_change", m.handler.RequestEmailChange)
			group.Post("/me/email/confirm_change", m.handler.ConfirmEmailChange)
			group.Post("/me/phone/request_change", m.handler.RequestPhoneChange)
			group.Post("/me/phone/confirm_change", m.handler.ConfirmPhoneChange)
			group.Post("/me/location/update", m.handler.UpdateLocation)
			group.Post("/me/upgrade", m.handler.Upgrade)
			group.Post("/me/settings/get", m.handler.GetMySettings)
			group.Post("/me/settings/update", m.handler.UpdateMySettings)
			group.Post("/me/delete", m.handler.DeleteAccount)
			group.Post("/me/follow", m.handler.Follow)
			group.Post("/me/unfollow", m.handler.Unfollow)
			group.Post("/me/block", m.handler.Block)
			group.Post("/me/unblock", m.handler.Unblock)
			group.Post("/me/list_blocked", m.handler.ListBlocked)
			group.Post("/me/set_remark", m.handler.SetRemark)
		})
	})
}
