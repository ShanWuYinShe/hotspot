package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	result, err := service.IssueSession(context.Background(), "user_alice")
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

func TestRequiredAuthRejectsLegacyStyleBearer(t *testing.T) {
	SetDefaultService(NewService(config.Config{AuthTokenSecret: "test-secret"}))
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	req := httptest.NewRequest(http.MethodPost, "/secure", http.NoBody)
	req.Header.Set("Authorization", "Bearer dXNlcl9sZWdhY3kuc2Vzc19sZWdhY3kuMTcxMjAwMDAwMC5zaWc")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" || body == "{}\n" {
		t.Fatal("expected api error body")
	}
}

func TestAdminAccessRejectsNonAdminPrincipal(t *testing.T) {
	handler := AdminAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/admin", http.NoBody)
	req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: "user_alice", SessionID: "sess"}))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" || body == "{}\n" {
		t.Fatal("expected api error body")
	}
}

// fakeBanChecker is a test BanChecker that bans a configured set of users.
type fakeBanChecker struct {
	banned map[string]bool
}

func (f *fakeBanChecker) IsBanned(_ context.Context, userID string) (bool, error) {
	return f.banned[userID], nil
}

// TestRequiredAuthRejectsBannedUserWith10201 verifies a banned user with a
// valid token gets 10201 + cleared cookies, never reaching the handler.
func TestRequiredAuthRejectsBannedUserWith10201(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(&fakeBanChecker{banned: map[string]bool{"user_banned": true}})
	SetDefaultService(service)

	result, err := service.IssueSession(context.Background(), "user_banned")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached for banned user")
	}))
	req := httptest.NewRequest(http.MethodPost, "/secure", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+result.AccessToken)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "10201") {
		t.Fatalf("expected 10201 in response body, got: %s", rr.Body.String())
	}
	// Web clients get auth cookies cleared (force logout).
	setCookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, AccessTokenCookieName) {
		t.Fatalf("expected access_token cookie to be cleared, got Set-Cookie: %s", setCookie)
	}
}

// TestRequiredAuthAllowsNonBannedUser verifies a non-banned user with a valid
// token passes through normally when a BanChecker is configured.
func TestRequiredAuthAllowsNonBannedUser(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(&fakeBanChecker{banned: map[string]bool{}})
	SetDefaultService(service)

	result, err := service.IssueSession(context.Background(), "user_ok")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	reached := false
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/secure", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+result.AccessToken)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !reached {
		t.Fatal("handler should be reached for non-banned user")
	}
}
