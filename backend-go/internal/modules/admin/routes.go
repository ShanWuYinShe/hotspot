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
	r.Route("/app_config", func(appConfig chi.Router) {
		appConfig.Post("/get", handler.GetAppConfig)
	})
}
