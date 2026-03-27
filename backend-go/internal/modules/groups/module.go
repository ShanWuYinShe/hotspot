package groups

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
	r.Route("/api/v1/groups", func(groupsRouter chi.Router) {
		groupsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/search", m.handler.Search)
			group.Post("/open_the_door", m.handler.OpenTheDoor)
			group.Post("/members/list", m.handler.ListMembers)
			group.Post("/members/search", m.handler.SearchMembers)
			group.Post("/announcements/list", m.handler.ListAnnouncements)
		})
		groupsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/update", m.handler.Update)
			group.Post("/dismiss", m.handler.Dismiss)
			group.Post("/me/list", m.handler.ListMyGroups)
			group.Post("/me/join", m.handler.JoinGroup)
			group.Post("/announcements/create", m.handler.CreateAnnouncement)
			group.Post("/announcements/update", m.handler.UpdateAnnouncement)
			group.Post("/announcements/delete", m.handler.DeleteAnnouncement)
			group.Post("/applications/list", m.handler.ListApplications)
			group.Post("/applications/approve", m.handler.ApproveApplication)
			group.Post("/applications/reject", m.handler.RejectApplication)
		})
	})
}
