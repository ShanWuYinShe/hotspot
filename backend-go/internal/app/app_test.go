package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"backend-go/internal/configtest"
)

// TestRegisterRoutes_registersAuthAndMapHandlers ensures that App.RegisterRoutes
// wires /api/v1/auth/login and /api/v1/map/canvas to real handlers that return
// HTTP 200 for a basic well-formed request.
func TestRegisterRoutes_registersAuthAndMapHandlers(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	cfg := configtest.Load(t)

	ctx := context.Background()
	application, err := NewApp(ctx, cfg, logger)
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}
	defer application.Close()

	r := chi.NewRouter()
	application.RegisterRoutes(r)

	// helper to perform a request and assert 200
	assertStatusOK := func(t *testing.T, method, path string, body string) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected %s %s to return 200, got %d, body=%s", method, path, w.Code, w.Body.String())
		}
	}

	t.Run("auth login route exists", func(t *testing.T) {
		// minimal valid login request body; underlying service may still be TODO,
		// but handler wiring and ApiResponse envelope should be exercised.
		body := `{"login_id":"test@example.com","password":"pw","client_type":"app"}`
		assertStatusOK(t, http.MethodPost, "/api/v1/auth/login", body)
	})

	t.Run("map canvas route exists", func(t *testing.T) {
		body := `{"viewport":{"bounds":{"north":0,"south":0,"east":0,"west":0}}}`
		assertStatusOK(t, http.MethodPost, "/api/v1/map/canvas", body)
	})
}
