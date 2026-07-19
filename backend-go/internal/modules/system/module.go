package system

import (
	"context"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory, source appconfig.Source) *Module {
	service := NewServiceWithFactory(factory, source)
	return &Module{handler: NewHandler(service)}
}

// SetHealthRedisPinger wires an optional redis liveness probe used by the
// health endpoint. Bootstrap calls this when a redis client is configured.
func (m *Module) SetHealthRedisPinger(p func(ctx context.Context) error) {
	if m == nil || m.handler == nil || m.handler.service == nil {
		return
	}
	m.handler.service.SetHealthRedisPinger(p)
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/system", func(systemRouter chi.Router) {
		systemRouter.Group(func(group chi.Router) {
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			MountRoutes(group, m.handler)
		})
	})
}
