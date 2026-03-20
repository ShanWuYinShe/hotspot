package auth

import (
	"github.com/go-chi/chi/v5"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(cfg config.Config, factory *persistence.Factory) *Module {
	authService := platformauth.NewServiceWithPersistence(cfg, factory)
	platformauth.SetDefaultService(authService)
	service := NewService(authService)
	return &Module{handler: NewHandler(service)}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/auth", func(authRouter chi.Router) {
		authRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/register", m.handler.Register)
			group.Post("/login", m.handler.Login)
			group.Post("/guest", m.handler.Guest)
			group.Post("/refresh_token", m.handler.RefreshToken)
			group.Post("/check_status", m.handler.CheckStatus)
			group.Post("/request_password_reset", m.handler.RequestPasswordReset)
			group.Post("/reset_password", m.handler.ResetPassword)
		})

		authRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/logout", m.handler.Logout)
			group.Post("/me/change_password", m.handler.ChangePassword)
			group.Post("/me/sessions/list", m.handler.ListSessions)
			group.Post("/me/sessions/revoke", m.handler.RevokeSession)
			group.Post("/me/sessions/revoke_all", m.handler.RevokeAllSessions)
		})
	})
}
