package users

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestUsersModuleMountsContactChangeRoutes(t *testing.T) {
	router := chi.NewRouter()
	NewModule(nil, nil).Mount(router)

	routes := map[string]bool{
		"/api/v1/users/email/request_change": false,
		"/api/v1/users/email/confirm_change": false,
		"/api/v1/users/phone/request_change": false,
		"/api/v1/users/phone/confirm_change": false,
	}
	if err := chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method != http.MethodPost {
			return nil
		}
		if _, ok := routes[route]; ok {
			routes[route] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk router: %v", err)
	}
	for route, found := range routes {
		if !found {
			t.Fatalf("expected route to be mounted: %s", route)
		}
	}
}
