package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// withAuthService 把装配期依赖注入请求 ctx（生产由根中间件
// platformauth.WithDependencies 注入；测试直调中间件时手动包裹）。
func withAuthService(req *http.Request, svc *Service) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), depsContextKey{}, requestDeps{service: svc}))
}

func TestRequiredAuthRejectsMissingToken(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := withAuthService(httptest.NewRequest(http.MethodPost, "/secure", http.NoBody), service)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" || body == "{}\n" {
		t.Fatal("expected api error body")
	}
}

func TestRequiredAuthFailsClosedWithoutDependencies(t *testing.T) {
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be reached without injected auth service")
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/secure", http.NoBody))
	if code := decodeAPIErrorCode(t, rr.Body.Bytes()); code != 10600 {
		t.Fatalf("expected 10600 fail-closed, got %d", code)
	}
}

func TestRequiredAuthRejectsLegacyStyleBearer(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	req := withAuthService(httptest.NewRequest(http.MethodPost, "/secure", http.NoBody), service)
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
// valid token gets httpx.CodeAccountBanned + cleared cookies, never reaching the handler.
func TestRequiredAuthRejectsBannedUserWith10201(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(&fakeBanChecker{banned: map[string]bool{"user_banned": true}})

	result, err := service.IssueSession(context.Background(), "user_banned")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached for banned user")
	}))
	req := withAuthService(httptest.NewRequest(http.MethodPost, "/secure", http.NoBody), service)
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

	result, err := service.IssueSession(context.Background(), "user_ok")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	reached := false
	handler := RequiredAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := withAuthService(httptest.NewRequest(http.MethodPost, "/secure", http.NoBody), service)
	req.Header.Set("Authorization", "Bearer "+result.AccessToken)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if !reached {
		t.Fatal("handler should be reached for non-banned user")
	}
}

func decodeAPIErrorCode(t *testing.T, body []byte) httpx.Code {
	t.Helper()
	var envelope struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode api error: %v body=%s", err, body)
	}
	return envelope.Code
}
