package auth

import "github.com/go-chi/chi/v5"

// MountPublicRoutes mounts the anonymous account-lifecycle endpoints on the
// /public/v1/auth subtree (spec api_spec_public.md).
func MountPublicRoutes(r chi.Router, handler *Handler) {
	r.Post("/register", handler.Register)
	r.Post("/signin", handler.Login)
	r.Post("/guest", handler.Guest)
	r.Post("/send_register_code", handler.SendRegisterCode)
	r.Post("/refresh_token", handler.RefreshToken)
	r.Post("/forgot_password", handler.RequestPasswordReset)
	r.Post("/reset_password", handler.ResetPassword)
}

func MountAuthenticatedRoutes(r chi.Router, handler *Handler) {
	r.Post("/session", handler.GetStatus)
	r.Post("/signout", handler.Logout)
	r.Post("/change_password", handler.ChangePassword)
	r.Post("/fetch_sessions", handler.ListSessions)
	r.Post("/revoke_session", handler.RevokeSession)
	r.Post("/revoke_other_sessions", handler.RevokeAllSessions)
}
