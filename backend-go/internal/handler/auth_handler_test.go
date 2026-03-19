package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"backend-go/internal/api"
	"backend-go/internal/service"
	"backend-go/internal/types"
)

// fakeAuthService 实现 service.AuthService，用于 Handler 层测试，避免真实 DB/JWT 依赖。
type fakeAuthService struct {
	registerTokens       *service.AuthTokens
	loginTokens          *service.AuthTokens
	guestTokens          *service.AuthTokens
	refreshTokens        *service.AuthTokens
	checkStatusResult    *types.AuthStatus
	checkStatusLastToken string

	refreshCalled bool
	logoutCalled  bool
}

func newFakeAuthService() *fakeAuthService {
	now := time.Now().UTC()
	defaultTokens := &service.AuthTokens{
		UserID:                "user-1",
		AccessToken:           "access-token",
		RefreshToken:          "refresh-token",
		AccessTokenExpiresAt:  now.Add(time.Minute),
		RefreshTokenExpiresAt: now.Add(24 * time.Hour),
	}
	return &fakeAuthService{
		registerTokens:    defaultTokens,
		loginTokens:       defaultTokens,
		guestTokens:       defaultTokens,
		refreshTokens:     defaultTokens,
		checkStatusResult: &types.AuthStatus{IsValid: true},
	}
}

func (f *fakeAuthService) Register(_ context.Context, req types.RegisterRequest) (*service.AuthTokens, error) {
	return f.registerTokens, nil
}

func (f *fakeAuthService) Login(_ context.Context, req types.LoginRequest) (*service.AuthTokens, error) {
	return f.loginTokens, nil
}

func (f *fakeAuthService) Logout(_ context.Context) error {
	f.logoutCalled = true
	return nil
}

func (f *fakeAuthService) Guest(_ context.Context, req types.GuestRequest) (*service.AuthTokens, error) {
	return f.guestTokens, nil
}

func (f *fakeAuthService) RefreshToken(_ context.Context, req types.RefreshTokenRequest, tokenStr string) (*service.AuthTokens, error) {
	f.refreshCalled = true
	return f.refreshTokens, nil
}

func (f *fakeAuthService) CheckStatus(_ context.Context, tokenStr string) (*types.AuthStatus, error) {
	f.checkStatusLastToken = tokenStr
	return f.checkStatusResult, nil
}

func (f *fakeAuthService) RequestPasswordReset(_ context.Context, req types.RequestPasswordResetRequest) error {
	return nil
}

func (f *fakeAuthService) ResetPassword(_ context.Context, req types.ResetPasswordRequest) (*service.AuthTokens, error) {
	return f.registerTokens, nil
}

func (f *fakeAuthService) ChangePassword(_ context.Context, userID string, req types.ChangePasswordRequest) error {
	return nil
}

func (f *fakeAuthService) ListSessions(_ context.Context, userID string) ([]types.SessionItem, error) {
	return nil, nil
}

func (f *fakeAuthService) RevokeSession(_ context.Context, userID string, req types.RevokeSessionRequest) error {
	return nil
}

func (f *fakeAuthService) RevokeAllSessions(_ context.Context, userID string) (*types.RevokeAllResult, error) {
	return &types.RevokeAllResult{RevokedCount: 0}, nil
}

var _ service.AuthService = (*fakeAuthService)(nil)

func newAuthRouter(svc service.AuthService) *chi.Mux {
	r := chi.NewRouter()
	h := NewAuthHandler(svc)
	RegisterAuthRoutes(r, h)
	return r
}

// --- /auth/register ---

func TestAuthHandler_Register_WebSetsCookiesAndEmptyBody(t *testing.T) {
	svc := newFakeAuthService()
	r := newAuthRouter(svc)

	body := types.RegisterRequest{
		LoginID:    "alice",
		Nickname:   "Alice",
		Password:   "secret",
		ClientType: types.ClientTypeWeb,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	if rs.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", rs.StatusCode)
	}
	var accessCookie, refreshCookie *http.Cookie
	for _, c := range rs.Cookies() {
		if c.Name == "access_token" {
			accessCookie = c
		}
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}
	if accessCookie == nil || accessCookie.Value == "" {
		t.Fatalf("expected access_token cookie to be set")
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatalf("expected refresh_token cookie to be set")
	}

	var resp api.Response[types.AuthTokenPairWeb]
	if err := json.NewDecoder(rs.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code=0, got %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatalf("expected data to be empty object, got nil")
	}
}

// --- /auth/login ---

func TestAuthHandler_Login_AppReturnsTokenPairInBody(t *testing.T) {
	svc := newFakeAuthService()
	svc.loginTokens = &service.AuthTokens{
		UserID:                "user-1",
		AccessToken:           "access-token",
		RefreshToken:          "refresh-token",
		AccessTokenExpiresAt:  time.Unix(1700000000, 0).UTC(),
		RefreshTokenExpiresAt: time.Unix(1700003600, 0).UTC(),
	}
	r := newAuthRouter(svc)

	body := types.LoginRequest{
		LoginID:    "alice",
		Password:   "secret",
		ClientType: types.ClientTypeApp,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	if rs.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", rs.StatusCode)
	}
	if len(rs.Cookies()) != 0 {
		t.Fatalf("expected no cookies for app client, got %d", len(rs.Cookies()))
	}
	var resp api.Response[types.AuthTokenPairApp]
	if err := json.NewDecoder(rs.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code=0, got %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatalf("expected data for app token pair, got nil")
	}
	if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		t.Fatalf("expected non-empty access/refresh tokens in body")
	}
}

// --- /auth/logout ---

func TestAuthHandler_Logout_ClearsCookiesForWebClient(t *testing.T) {
	svc := newFakeAuthService()
	r := newAuthRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "access-token"})
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-token", Path: "/api/v1/auth/refresh_token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	if rs.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", rs.StatusCode)
	}
	if !svc.logoutCalled {
		t.Fatalf("expected service.Logout to be called")
	}

	var accessCleared, refreshCleared bool
	for _, c := range rs.Cookies() {
		if c.Name == "access_token" {
			if c.MaxAge != 0 || c.Value != "" || c.Path != "/" {
				t.Fatalf("expected access_token cookie cleared with MaxAge=0 and empty value, got %+v", c)
			}
			accessCleared = true
		}
		if c.Name == "refresh_token" {
			if c.MaxAge != 0 || c.Value != "" || c.Path != "/api/v1/auth/refresh_token" {
				t.Fatalf("expected refresh_token cookie cleared with MaxAge=0 and empty value, got %+v", c)
			}
			refreshCleared = true
		}
	}
	if !accessCleared || !refreshCleared {
		t.Fatalf("expected both access_token and refresh_token cookies to be cleared")
	}

	var resp api.Response[struct{}]
	if err := json.NewDecoder(rs.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code=0, got %d", resp.Code)
	}
}

// --- /auth/refresh_token ---

func TestAuthHandler_RefreshToken_WebBodyRefreshTokenRejected(t *testing.T) {
	svc := newFakeAuthService()
	r := newAuthRouter(svc)

	body := types.RefreshTokenRequest{
		ClientType:   types.ClientTypeWeb,
		RefreshToken: ptrString("should-not-be-here"),
	}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh_token", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	var resp api.Response[struct{}]
	if err := json.NewDecoder(rs.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10000 {
		t.Fatalf("expected code=10000 for invalid web body, got %d", resp.Code)
	}
	if svc.refreshCalled {
		t.Fatalf("expected service.RefreshToken not to be called when body violates web rules")
	}
}

func TestAuthHandler_RefreshToken_WebMissingCookieReturns10100(t *testing.T) {
	svc := newFakeAuthService()
	r := newAuthRouter(svc)

	body := types.RefreshTokenRequest{ClientType: types.ClientTypeWeb}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh_token", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	var resp api.Response[struct{}]
	if err := json.NewDecoder(rs.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10100 {
		t.Fatalf("expected code=10100 when web refresh_token cookie missing, got %d", resp.Code)
	}
	if svc.refreshCalled {
		t.Fatalf("expected service.RefreshToken not to be called when cookie missing")
	}
}

func TestAuthHandler_RefreshToken_WebUsesCookieOnSuccess(t *testing.T) {
	svc := newFakeAuthService()
	r := newAuthRouter(svc)

	body := types.RefreshTokenRequest{ClientType: types.ClientTypeWeb}
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh_token", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-refresh"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	rs := w.Result()
	if rs.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", rs.StatusCode)
	}
	if !svc.refreshCalled {
		t.Fatalf("expected service.RefreshToken to be called when cookie present")
	}
	var accessCookie, refreshCookie *http.Cookie
	for _, c := range rs.Cookies() {
		if c.Name == "access_token" {
			accessCookie = c
		}
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}
	if accessCookie == nil || refreshCookie == nil {
		t.Fatalf("expected refreshed access_token and refresh_token cookies to be set")
	}
}

// --- /auth/check_status ---

func TestAuthHandler_CheckStatus_UsesAuthorizationHeaderWhenPresent(t *testing.T) {
	svc := newFakeAuthService()
	svc.checkStatusResult = &types.AuthStatus{IsValid: true}
	r := newAuthRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/check_status", nil)
	req.Header.Set("Authorization", "Bearer my-access-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if svc.checkStatusLastToken != "my-access-token" {
		t.Fatalf("expected CheckStatus to receive token from Authorization header, got %q", svc.checkStatusLastToken)
	}
	var resp api.Response[types.AuthStatus]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data == nil || !resp.Data.IsValid {
		t.Fatalf("expected success response with is_valid=true, got %+v", resp)
	}
}

func TestAuthHandler_CheckStatus_FallsBackToCookieAndEmptyToken(t *testing.T) {
	svc := newFakeAuthService()
	svc.checkStatusResult = &types.AuthStatus{IsValid: false}
	r := newAuthRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/check_status", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-access-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if svc.checkStatusLastToken != "cookie-access-token" {
		t.Fatalf("expected CheckStatus to use access_token cookie when Authorization missing, got %q", svc.checkStatusLastToken)
	}
	var resp api.Response[types.AuthStatus]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code=0, got %d", resp.Code)
	}
	if resp.Data == nil || resp.Data.IsValid {
		t.Fatalf("expected is_valid=false from fake service, got %+v", resp.Data)
	}
}

// --- helpers ---

func ptrString(s string) *string { return &s }
