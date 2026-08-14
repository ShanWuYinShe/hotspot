package auth

import "github.com/go-chi/chi/v5"

func MountRoutes(r chi.Router, handler *Handler) {
	r.Post("/register", handler.Register)
	r.Post("/login", handler.Login)
	r.Post("/guest", handler.Guest)
	r.Post("/refresh_token", handler.RefreshToken)
	r.Post("/get_status", handler.GetStatus)
	r.Post("/request_password_reset", handler.RequestPasswordReset)
	r.Post("/reset_password", handler.ResetPassword)
}

func MountAuthenticatedRoutes(r chi.Router, handler *Handler) {
	r.Post("/logout", handler.Logout)
	r.Post("/change_password", handler.ChangePassword)
	r.Post("/list_sessions", handler.ListSessions)
	r.Post("/revoke_session", handler.RevokeSession)
	r.Post("/revoke_all_sessions", handler.RevokeAllSessions)
}
