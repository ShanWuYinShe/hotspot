package auth

import "github.com/go-chi/chi/v5"

func Routes(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/login", handler.Login)
	r.Post("/refresh_token", handler.RefreshToken)
	r.Post("/check_status", handler.CheckStatus)
	return r
}

func AuthenticatedRoutes(handler *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/logout", handler.Logout)
	r.Post("/me/sessions/list", handler.ListSessions)
	r.Post("/me/sessions/revoke", handler.RevokeSession)
	r.Post("/me/sessions/revoke_all", handler.RevokeAllSessions)
	return r
}
