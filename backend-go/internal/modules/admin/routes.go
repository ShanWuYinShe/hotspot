package admin

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Post("/rbac/get_me", handler.GetMe)
	r.Post("/app_config/get", handler.GetAppConfig)
}
