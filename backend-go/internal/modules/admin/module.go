package admin

import (
	"github.com/go-chi/chi/v5"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct{ handler *Handler }

func NewModule(cfg config.Config) *Module {
	return &Module{handler: NewHandler(NewService(appconfig.NewStaticSource(cfg)))}
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/admin", func(ar chi.Router) {
		ar.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(platformauth.AdminAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/rbac/get_me", m.handler.GetMe)
			group.Post("/app_config/get", m.handler.GetAppConfig)
		})
	})
}
