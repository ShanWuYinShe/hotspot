package mapmodule

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/persistence"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory) *Module {
	repo := NewPostgresRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithRepo(repo))}
}

// Mount implements modulekit.Module; the map canvas is public-only, so this
// is a no-op (see MountPublic).
func (m *Module) Mount(r chi.Router) {}

// The map canvas is public-only (spec api_spec_public.md): it moved to
// /public/v1/map/canvas with the node split, so there is no /api/v1 mount.
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/map", func(mapRouter chi.Router) {
		mapRouter.Post("/canvas", m.handler.Canvas)
	})
}
