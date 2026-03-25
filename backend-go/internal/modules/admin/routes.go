package admin

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Route("/rbac", func(rbac chi.Router) {
		rbac.Post("/get_me", handler.GetMe)
		rbac.Post("/list_permissions", handler.ListPermissions)
		rbac.Route("/roles", func(roles chi.Router) {
			roles.Post("/list", handler.ListRoles)
			roles.Post("/create", handler.CreateRole)
			roles.Post("/update", handler.UpdateRole)
			roles.Post("/delete", handler.DeleteRole)
			roles.Post("/set_permissions", handler.SetRolePermissions)
		})
		rbac.Route("/admins", func(admins chi.Router) {
			admins.Post("/list", handler.ListAdmins)
			admins.Post("/grant", handler.GrantAdmin)
			admins.Post("/revoke", handler.RevokeAdmin)
		})
	})
	r.Route("/audit", func(audit chi.Router) {
		audit.Post("/list", handler.ListAuditLogs)
		audit.Post("/open_the_door", handler.OpenAuditLog)
	})
	r.Route("/app_config", func(appConfig chi.Router) {
		appConfig.Post("/get", handler.GetAppConfig)
		appConfig.Post("/update_upload", handler.UpdateUpload)
		appConfig.Post("/update_defaults", handler.UpdateDefaults)
		appConfig.Post("/update_rate_limits", handler.UpdateRateLimits)
		appConfig.Post("/update_response_defaults", handler.UpdateResponseDefaults)
		appConfig.Post("/update_features", handler.UpdateFeatures)
		appConfig.Post("/update_limits", handler.UpdateLimits)
		appConfig.Post("/update_contact", handler.UpdateContact)
	})
	r.Route("/users", func(users chi.Router) {
		users.Post("/search", handler.SearchUsers)
		users.Post("/open_the_door", handler.OpenAdminUser)
		users.Post("/ban", handler.BanUser)
		users.Post("/unban", handler.UnbanUser)
	})
	r.Route("/groups", func(groups chi.Router) {
		groups.Post("/search", handler.SearchGroups)
		groups.Post("/open_the_door", handler.OpenAdminGroup)
		groups.Post("/dismiss", handler.DismissGroup)
	})
	r.Route("/reports", func(reports chi.Router) {
		reports.Post("/search", handler.SearchReports)
		reports.Post("/handle", handler.HandleReport)
	})
	r.Route("/tags", func(tags chi.Router) {
		tags.Post("/search", handler.SearchTags)
		tags.Post("/delete", handler.DeleteTag)
	})
	r.Route("/announcements", func(announcements chi.Router) {
		announcements.Post("/search", handler.SearchAnnouncements)
		announcements.Post("/create", handler.CreateAnnouncement)
		announcements.Post("/update", handler.UpdateAnnouncement)
		announcements.Post("/delete", handler.DeleteAnnouncement)
	})
}
