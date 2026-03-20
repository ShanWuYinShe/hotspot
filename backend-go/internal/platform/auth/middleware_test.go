package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

func TestRequiredAuthRejectsMissingToken(t *testing.T) {
	SetDefaultService(NewService(config.Config{AuthTokenSecret: "test-secret"}))
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/secure", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" || body == "{}\n" {
		t.Fatal("expected api error body")
	}
}

func TestOptionalAuthInjectsPrincipalForValidBearer(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	SetDefaultService(service)
	result, err := service.IssueSession("user_alice")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	token := result.AccessToken

	handler := OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || principal.UserID != "user_alice" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/optional", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}
