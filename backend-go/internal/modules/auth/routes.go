package auth

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Post("/register", handler.Register)
	r.Post("/login", handler.Login)
	r.Post("/guest", handler.Guest)
	r.Post("/refresh_token", handler.RefreshToken)
	r.Post("/check_status", handler.CheckStatus)
	r.Post("/request_password_reset", handler.RequestPasswordReset)
	r.Post("/reset_password", handler.ResetPassword)
}

func MountAuthenticatedRoutes(r chi.Router, handler *Handler) {
	r.Post("/logout", handler.Logout)
	r.Post("/me/change_password", handler.ChangePassword)
	r.Post("/me/sessions/list", handler.ListSessions)
	r.Post("/me/sessions/revoke", handler.RevokeSession)
	r.Post("/me/sessions/revoke_all", handler.RevokeAllSessions)
}
