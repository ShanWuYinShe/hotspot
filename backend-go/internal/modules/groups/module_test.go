package groups

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGroupsModuleMountsSpecListBannedRoute(t *testing.T) {
	router := chi.NewRouter()
	NewModule(nil).Mount(router)

	var hasSpecRoute bool
	var hasLegacyRoute bool
	if err := chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method != http.MethodPost {
			return nil
		}
		switch route {
		case "/api/v1/groups/members/list_banned":
			hasSpecRoute = true
		case "/api/v1/groups/members/banned/list":
			hasLegacyRoute = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk router: %v", err)
	}
	if !hasSpecRoute {
		t.Fatal("expected spec route to be mounted")
	}
	if hasLegacyRoute {
		t.Fatal("did not expect legacy route to remain mounted")
	}
}
