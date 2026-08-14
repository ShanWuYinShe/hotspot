package admin

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Route("/rbac", func(rbac chi.Router) {
		rbac.Post("/me", handler.GetMe)
		rbac.Route("/permissions", func(perms chi.Router) {
			perms.Post("/fetch", handler.ListPermissions)
		})
		rbac.Route("/roles", func(roles chi.Router) {
			roles.Post("/fetch", handler.ListRoles)
			roles.Post("/create", handler.CreateRole)
			roles.Post("/update", handler.UpdateRole)
			roles.Post("/delete", handler.DeleteRole)
			roles.Post("/set_permissions", handler.SetRolePermissions)
		})
		rbac.Route("/admins", func(admins chi.Router) {
			admins.Post("/fetch", handler.ListAdmins)
			admins.Post("/award", handler.GrantAdmin)
			admins.Post("/revoke", handler.RevokeAdmin)
		})
	})
	r.Route("/audit", func(audit chi.Router) {
		audit.Post("/fetch", handler.ListAuditLogs)
	})
	// /admin/v1/app_config 是名词化端点（无动词段），直接注册完整路径，
	// 避免 chi 子路由 "/" 模式的尾斜杠歧义。
	r.Post("/app_config", handler.GetAppConfig)
	r.Route("/app_config", func(appConfig chi.Router) {
		appConfig.Post("/update", handler.UpdateAppConfig)
	})
	r.Route("/users", func(users chi.Router) {
		users.Post("/search", handler.SearchUsers)
		users.Post("/open_the_door", handler.OpenAdminUser)
		users.Post("/set_ban", handler.SetUserBan)
	})
	r.Route("/groups", func(groups chi.Router) {
		groups.Post("/search", handler.SearchGroups)
		groups.Post("/open_the_door", handler.OpenAdminGroup)
		groups.Post("/dismiss", handler.DismissGroup)
	})
	r.Route("/reports", func(reports chi.Router) {
		reports.Post("/fetch", handler.SearchReports)
		reports.Post("/judge", handler.HandleReport)
	})
	r.Route("/tags", func(tags chi.Router) {
		tags.Post("/search", handler.SearchTags)
		tags.Post("/delete", handler.DeleteTag)
	})
	r.Route("/announcements", func(announcements chi.Router) {
		announcements.Post("/fetch", handler.ListAdminAnnouncements)
		announcements.Post("/create", handler.CreateAnnouncement)
		announcements.Post("/update", handler.UpdateAnnouncement)
		announcements.Post("/delete", handler.DeleteAnnouncement)
	})
}
