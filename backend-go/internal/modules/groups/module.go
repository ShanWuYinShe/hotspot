package groups

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notifications"
	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory, actorProvider actor.Provider) *Module {
	repo := NewPersistenceRepository(factory)
	notificationService := notifications.NewServiceWithRepo(notifications.NewPersistenceRepository(factory))
	return &Module{handler: NewHandler(NewServiceWithDeps(repo, notificationService, actorProvider))}
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
			group.Post("/me/transfer_ownership", m.handler.TransferOwnership)
			group.Post("/me/update_settings", m.handler.UpdateMySettings)
			group.Post("/me/leave", m.handler.LeaveGroup)
			group.Post("/me/applications/list", m.handler.ListMyApplications)
			group.Post("/me/applications/cancel", m.handler.CancelApplication)
			group.Post("/me/invites/list", m.handler.ListMyInvites)
			group.Post("/me/invites/accept", m.handler.AcceptInvite)
			group.Post("/me/invites/reject", m.handler.RejectInvite)
			group.Post("/members/invite", m.handler.InviteMember)
			group.Post("/members/invites/list", m.handler.ListSentInvites)
			group.Post("/members/invites/revoke", m.handler.RevokeInvite)
			group.Post("/members/set_role", m.handler.SetMemberRole)
			group.Post("/members/mute", m.handler.MuteMember)
			group.Post("/members/unmute", m.handler.UnmuteMember)
			group.Post("/members/ban", m.handler.BanMember)
			group.Post("/members/unban", m.handler.UnbanMember)
			group.Post("/members/list_banned", m.handler.ListBannedMembers)
			group.Post("/members/remove_member", m.handler.RemoveMember)
			group.Post("/announcements/create", m.handler.CreateAnnouncement)
			group.Post("/announcements/update", m.handler.UpdateAnnouncement)
			group.Post("/announcements/delete", m.handler.DeleteAnnouncement)
			group.Post("/applications/list", m.handler.ListApplications)
			group.Post("/applications/approve", m.handler.ApproveApplication)
			group.Post("/applications/reject", m.handler.RejectApplication)
		})
	})
}
