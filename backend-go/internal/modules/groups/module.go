package groups

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
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

func NewModule(factory *persistence.Factory, actorProvider actor.Provider) *Module {
	repo := NewPostgresRepository(factory)
	notificationService := notify.NewServiceWithRepo(notify.NewPostgresRepository(factory))
	svc := NewServiceWithDeps(repo, notificationService, actorProvider)
	return &Module{handler: NewHandler(svc), Service: svc}
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
			group.Post("/list_my", m.handler.ListMyGroups)
			group.Post("/join", m.handler.JoinGroup)
			group.Post("/transfer_ownership", m.handler.TransferOwnership)
			group.Post("/update_preferences", m.handler.UpdatePreferences)
			group.Post("/leave", m.handler.LeaveGroup)
			group.Post("/list_applications", m.handler.ListMyApplications)
			group.Post("/cancel_application", m.handler.CancelApplication)
			group.Post("/invites/received/list", m.handler.ListMyInvites)
			group.Post("/invites/received/accept", m.handler.AcceptInvite)
			group.Post("/invites/received/reject", m.handler.RejectInvite)
			group.Post("/members/invite", m.handler.InviteMember)
			group.Post("/invites/sent/list", m.handler.ListSentInvites)
			group.Post("/invites/sent/revoke", m.handler.RevokeInvite)
			group.Post("/members/set_role", m.handler.SetMemberRole)
			group.Post("/members/set_mute", m.handler.SetMute)
			group.Post("/members/set_ban", m.handler.SetMemberBan)
			group.Post("/members/list_banned", m.handler.ListBannedMembers)
			group.Post("/members/remove", m.handler.RemoveMember)
			group.Post("/announcements/create", m.handler.CreateAnnouncement)
			group.Post("/announcements/update", m.handler.UpdateAnnouncement)
			group.Post("/announcements/delete", m.handler.DeleteAnnouncement)
			group.Post("/applications/list", m.handler.ListApplications)
			group.Post("/applications/approve", m.handler.ApproveApplication)
			group.Post("/applications/reject", m.handler.RejectApplication)
		})
	})
}
