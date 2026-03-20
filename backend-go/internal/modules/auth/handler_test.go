package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
)

func newTestHandler() *Handler {
	service := NewService(platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"}))
	return NewHandler(service)
}

func TestRegisterRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"login_id":"alice","nickname":"Alice","password":"secret","client_type":"app"}`))
	rr := httptest.NewRecorder()

	handler.Register(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestLoginAppReturnsTokenPair(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	var resp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCheckStatusReturnsFalseWithoutPrincipal(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/check_status", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.CheckStatus(rr, req)

	var resp struct {
		Code int        `json:"code"`
		Data AuthStatus `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.IsValid {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCheckStatusReturnsTrueWithPrincipal(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/check_status", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice"})
	ctx = platformauth.WithAccessExpiry(ctx, time.Now().UTC().Add(time.Minute))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.CheckStatus(rr, req)

	var resp struct {
		Code int        `json:"code"`
		Data AuthStatus `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || !resp.Data.IsValid || resp.Data.ExpiresAt == "" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGuestWithIdempotencyKeyReturnsTokenPair(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"nickname":"Guest","client_type":"app"}`))
	req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.Guest(rr, req)
	rr2 := httptest.NewRecorder()
	dupe := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"nickname":"Guest","client_type":"app"}`))
	dupe.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	handler.Guest(rr2, dupe)

	var resp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var resp2 struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	_ = json.NewDecoder(rr2.Body).Decode(&resp2)
	if resp.Code != 0 || resp.Data.UserID == "" || resp.Data.UserID != resp2.Data.UserID {
		t.Fatalf("unexpected response: %#v %#v", resp, resp2)
	}
}

func TestRefreshTokenAppReturnsNewTokenPair(t *testing.T) {
	handler := newTestHandler()
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)
	var loginResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(loginRR.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(`{"client_type":"app","refresh_token":"`+loginResp.Data.RefreshToken+`"}`))
	refreshRR := httptest.NewRecorder()
	handler.RefreshToken(refreshRR, refreshReq)

	var refreshResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(refreshRR.Body).Decode(&refreshResp); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if refreshResp.Code != 0 || refreshResp.Data.AccessToken == "" || refreshResp.Data.RefreshToken == "" {
		t.Fatalf("unexpected refresh response: %#v", refreshResp)
	}
}

func TestListSessionsReturnsCurrentSession(t *testing.T) {
	handler := newTestHandler()
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)
	var loginResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(loginRR.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	sessionID := handler.service.auth.UnsafeFirstSessionIDForUser(loginResp.Data.UserID)
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	req := httptest.NewRequest(http.MethodPost, "/me/sessions/list", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: loginResp.Data.UserID, SessionID: sessionID})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ListSessions(rr, req)

	var resp struct {
		Code int         `json:"code"`
		Data SessionList `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) == 0 || !resp.Data.Items[0].IsCurrent {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestChangePasswordRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/me/change_password", strings.NewReader(`{"old_password":"old","new_password":"new"}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ChangePassword(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestLogoutClearsCookies(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) < 3 {
		t.Fatalf("expected auth cookies to be cleared, got %d", len(cookies))
	}
}
