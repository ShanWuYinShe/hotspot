package system

import "github.com/go-chi/chi/v5"

// MountPublicRoutes mounts the public-only system endpoints (spec
// api_spec_public.md) on the /public/v1/system subtree.
func MountPublicRoutes(r chi.Router, handler *Handler) {
	r.Post("/meta", handler.GetMeta)
	r.Post("/health", handler.Health)
	r.Post("/announcements/fetch", handler.ListAnnouncements)
	r.Post("/client_config", handler.GetClientConfig)
}
