package system

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Post("/get_meta", handler.GetMeta)
	r.Post("/health", handler.Health)
	r.Post("/announcements/list", handler.ListAnnouncements)
	r.Post("/get_client_config", handler.GetClientConfig)
}
