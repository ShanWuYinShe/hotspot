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

func NewModule(factory *persistence.Factory, actorProvider actor.Provider, notificationService *notify.Service) *Module {
	repo := NewPostgresRepository(factory)
	svc := NewServiceWithDeps(repo, notificationService, actorProvider)
	// 守卫子树角色解析器自接线：groups 自身的 Service.RoleOf 实现。
	SetGroupRoleResolver(svc)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/groups", func(groupsRouter chi.Router) {
		// 成员路径（必须登录）
		groupsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/search", m.handler.Search)
			group.Post("/open_the_door", m.handler.OpenTheDoor)
			group.Post("/members/fetch", m.handler.ListMembers)
			group.Post("/members/search", m.handler.SearchMembers)
			group.Post("/create", m.handler.Create)
			group.Post("/enter", m.handler.JoinGroup)
			group.Post("/exit", m.handler.LeaveGroup)
			group.Post("/fetch_joined", m.handler.ListMyGroups)
			group.Post("/update_preferences", m.handler.UpdatePreferences)
			group.Post("/applications/fetch_submitted", m.handler.ListMyApplications)
			group.Post("/applications/cancel", m.handler.CancelApplication)
			group.Post("/invites/fetch_received", m.handler.ListMyInvites)
			group.Post("/invites/accept", m.handler.AcceptInvite)
			group.Post("/invites/reject", m.handler.RejectInvite)
		})
		// 群管理守卫子树（/api/v1/groups/admin/*：调用者必须是目标群 owner/admin）
		groupsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Route("/admin", func(admin chi.Router) {
				admin.Use(groupAdminGuard(false))
				admin.Post("/update", m.handler.Update)
				admin.Post("/settings", m.handler.GetSettings)
				admin.Post("/invites/create", m.handler.InviteMember)
				admin.Post("/invites/fetch_sent", m.handler.ListSentInvites)
				admin.Post("/invites/revoke", m.handler.RevokeInvite)
				admin.Post("/members/fetch", m.handler.ListAdminMembers)
				admin.Post("/members/set_role", m.handler.SetMemberRole)
				admin.Post("/members/set_mute", m.handler.SetMute)
				admin.Post("/members/set_ban", m.handler.SetMemberBan)
				admin.Post("/members/fetch_banned", m.handler.ListBannedMembers)
				admin.Post("/members/remove", m.handler.RemoveMember)
				admin.Post("/applications/fetch", m.handler.ListApplications)
				admin.Post("/applications/approve", m.handler.ApproveApplication)
				admin.Post("/applications/reject", m.handler.RejectApplication)
				admin.Post("/announcements/create", m.handler.CreateAnnouncement)
				admin.Post("/announcements/update", m.handler.UpdateAnnouncement)
				admin.Post("/announcements/delete", m.handler.DeleteAnnouncement)
				admin.Post("/messages/recall", m.handler.RecallGroupMessageAsAdmin)
			})
			// owner 专属端点（守卫内再判 owner）
			group.Group(func(owner chi.Router) {
				owner.Use(groupAdminGuard(true))
				owner.Post("/admin/dismiss", m.handler.Dismiss)
				owner.Post("/admin/transfer_ownership", m.handler.TransferOwnership)
			})
		})
	})
}

// MountPublic mounts the anonymous groups endpoints on the /public/v1 node
// (spec api_spec_public.md). Group announcements are public-only — the
// /api/v1 mirror was removed with the node split.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/groups", func(groupsRouter chi.Router) {
		groupsRouter.Post("/search", m.handler.Search)
		groupsRouter.Post("/open_the_door", m.handler.OpenTheDoor)
		groupsRouter.Post("/members/fetch", m.handler.ListMembers)
		groupsRouter.Post("/members/search", m.handler.SearchMembers)
		groupsRouter.Route("/announcements", func(ann chi.Router) {
			ann.Post("/fetch", m.handler.ListAnnouncements)
		})
	})
}
