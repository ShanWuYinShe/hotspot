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
	// 守卫子树角色解析器自接线：groups 自身的 Service.RoleOf 实现。
	SetGroupRoleResolver(svc)
	return &Module{handler: NewHandler(svc), Service: svc}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/groups", func(groupsRouter chi.Router) {
		// 公开/成员路径（可选登录）
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
		// 成员路径（必须登录）
		groupsRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/join", m.handler.JoinGroup)
			group.Post("/leave", m.handler.LeaveGroup)
			group.Post("/list_joined", m.handler.ListMyGroups)
			group.Post("/update_preferences", m.handler.UpdatePreferences)
			group.Post("/applications/list_submitted", m.handler.ListMyApplications)
			group.Post("/applications/cancel", m.handler.CancelApplication)
			group.Post("/invites/list_received", m.handler.ListMyInvites)
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
				admin.Post("/get_settings", m.handler.GetSettings)
				admin.Post("/invites/create", m.handler.InviteMember)
				admin.Post("/invites/list_sent", m.handler.ListSentInvites)
				admin.Post("/invites/revoke", m.handler.RevokeInvite)
				admin.Post("/members/list", m.handler.ListAdminMembers)
				admin.Post("/members/set_role", m.handler.SetMemberRole)
				admin.Post("/members/set_mute", m.handler.SetMute)
				admin.Post("/members/set_ban", m.handler.SetMemberBan)
				admin.Post("/members/list_banned", m.handler.ListBannedMembers)
				admin.Post("/members/remove", m.handler.RemoveMember)
				admin.Post("/applications/list", m.handler.ListApplications)
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
