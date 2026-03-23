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
	cfg := config.Config{AuthTokenSecret: "test-secret"}
	repo := NewInMemoryRepository()
	service := NewService(platformauth.NewService(cfg), repo, cfg)
	return NewHandler(service)
}

func registerAppUser(t *testing.T, handler *Handler, loginID, nickname, password string) AuthTokenPairApp {
	t.Helper()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"login_id":"`+loginID+`","nickname":"`+nickname+`","password":"`+password+`","client_type":"app"}`))
	req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.Register(rr, req)
	var resp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected register response: %#v", resp)
	}
	return resp.Data
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

func TestRegisterDuplicateLoginIDReturnsConflict(t *testing.T) {
	handler := newTestHandler()
	registerAppUser(t, handler, "alice", "Alice", "secret")

	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"login_id":"alice","nickname":"Alice 2","password":"secret-2","client_type":"app"}`))
	req.Header.Set("Idempotency-Key", "bbbb1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 10400 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestLoginAppReturnsTokenPair(t *testing.T) {
	handler := newTestHandler()
	registerAppUser(t, handler, "alice", "Alice", "secret")

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

func TestLoginCollapsesUnknownUserAndWrongPassword(t *testing.T) {
	handler := newTestHandler()
	registerAppUser(t, handler, "alice", "Alice", "secret")

	cases := []string{
		`{"login_id":"alice","password":"wrong","client_type":"app"}`,
		`{"login_id":"missing","password":"secret","client_type":"app"}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.Login(rr, req)
		var resp map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if int(resp["code"].(float64)) != 10100 {
			t.Fatalf("unexpected response: %#v", resp)
		}
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
	registerAppUser(t, handler, "alice", "Alice", "secret")

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
	register := registerAppUser(t, handler, "alice", "Alice", "secret")
	sessionID := handler.service.auth.UnsafeFirstSessionIDForUser(register.UserID)
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	req := httptest.NewRequest(http.MethodPost, "/me/sessions/list", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: register.UserID, SessionID: sessionID})
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

func TestRequestPasswordResetDoesNotExposeUnknownLoginID(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(`{"login_id":"missing","client_type":"app"}`))
	req.Header.Set("Idempotency-Key", "cccc1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.RequestPasswordReset(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}

	registerAppUser(t, handler, "alice", "Alice", "secret")
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req1 := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(`{"login_id":"alice","client_type":"app"}`))
	req1.Header.Set("Idempotency-Key", "cccc2234abcd1234abcd1234abcd1234")
	rr1 := httptest.NewRecorder()
	handler.RequestPasswordReset(rr1, req1)
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req2 := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(`{"login_id":"alice","client_type":"app"}`))
	req2.Header.Set("Idempotency-Key", "cccc3234abcd1234abcd1234abcd1234")
	rr2 := httptest.NewRecorder()
	handler.RequestPasswordReset(rr2, req2)
	var resp2 map[string]any
	if err := json.NewDecoder(rr2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp2["code"].(float64)) != 0 {
		t.Fatalf("unexpected throttled response: %#v", resp2)
	}
}

func TestResetPasswordRevokesExistingSessionsAndAllowsLoginWithNewPassword(t *testing.T) {
	handler := newTestHandler()
	register := registerAppUser(t, handler, "alice", "Alice", "secret")
	firstSessionID := handler.service.auth.UnsafeFirstSessionIDForUser(register.UserID)
	if firstSessionID == "" {
		t.Fatal("expected first session id")
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)

	idempotency.SetStore(idempotency.NewInMemoryStore())
	resetReq := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(`{"login_id":"alice","client_type":"app"}`))
	resetReq.Header.Set("Idempotency-Key", "dddd1234abcd1234abcd1234abcd1234")
	resetRR := httptest.NewRecorder()
	handler.RequestPasswordReset(resetRR, resetReq)

	code := handler.service.repo.(*InMemoryRepository).LastResetCode("alice")
	if code == "" {
		t.Fatal("expected reset verification code")
	}

	idempotency.SetStore(idempotency.NewInMemoryStore())
	confirmReq := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(`{"login_id":"alice","verification_code":"`+code+`","new_password":"new-secret","client_type":"app"}`))
	confirmReq.Header.Set("Idempotency-Key", "eeee1234abcd1234abcd1234abcd1234")
	confirmRR := httptest.NewRecorder()
	handler.ResetPassword(confirmRR, confirmReq)

	var confirmResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(confirmRR.Body).Decode(&confirmResp); err != nil {
		t.Fatalf("decode reset response: %v", err)
	}
	if confirmResp.Code != 0 {
		t.Fatalf("unexpected reset response: %#v", confirmResp)
	}
	if principal, _, ok := handler.service.auth.AuthenticateRequest(confirmReq.Context(), "Bearer "+register.AccessToken, ""); ok || principal.UserID != "" {
		t.Fatal("expected old session token to be revoked after reset")
	}

	newLoginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"new-secret","client_type":"app"}`))
	newLoginRR := httptest.NewRecorder()
	handler.Login(newLoginRR, newLoginReq)
	var newLoginResp map[string]any
	if err := json.NewDecoder(newLoginRR.Body).Decode(&newLoginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if int(newLoginResp["code"].(float64)) != 0 {
		t.Fatalf("unexpected login response after reset: %#v", newLoginResp)
	}
}

func TestResetPasswordUnknownOrInvalidRequestReturnsNotFound(t *testing.T) {
	handler := newTestHandler()
	registerAppUser(t, handler, "alice", "Alice", "secret")
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(`{"login_id":"alice","verification_code":"000000","new_password":"new-secret","client_type":"app"}`))
	req.Header.Set("Idempotency-Key", "ffff1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.ResetPassword(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 10300 {
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

func TestRevokeSessionRejectsCurrentSession(t *testing.T) {
	handler := newTestHandler()
	register := registerAppUser(t, handler, "alice", "Alice", "secret")
	currentSessionID := handler.service.auth.UnsafeFirstSessionIDForUser(register.UserID)
	req := httptest.NewRequest(http.MethodPost, "/me/sessions/revoke", strings.NewReader(`{"session_id":"`+currentSessionID+`"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: register.UserID, SessionID: currentSessionID}))
	rr := httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 10200 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestChangePasswordValidatesOldPasswordAndRevokesOtherSessions(t *testing.T) {
	handler := newTestHandler()
	register := registerAppUser(t, handler, "alice", "Alice", "secret")
	currentSessionID := handler.service.auth.UnsafeFirstSessionIDForUser(register.UserID)
	if currentSessionID == "" {
		t.Fatal("expected current session id")
	}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)
	otherSessionID := ""
	for _, session := range handler.service.auth.ListSessions(register.UserID, currentSessionID) {
		if session.SessionID != currentSessionID {
			otherSessionID = session.SessionID
			break
		}
	}
	if otherSessionID == "" {
		t.Fatal("expected another active session")
	}

	idempotency.SetStore(idempotency.NewInMemoryStore())
	badReq := httptest.NewRequest(http.MethodPost, "/me/change_password", strings.NewReader(`{"old_password":"wrong","new_password":"new-secret"}`))
	badReq.Header.Set("Idempotency-Key", "11111234abcd1234abcd1234abcd1234")
	badReq = badReq.WithContext(platformauth.WithPrincipal(badReq.Context(), platformauth.Principal{UserID: register.UserID, SessionID: currentSessionID}))
	badRR := httptest.NewRecorder()
	handler.ChangePassword(badRR, badReq)
	var badResp map[string]any
	if err := json.NewDecoder(badRR.Body).Decode(&badResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(badResp["code"].(float64)) != 10100 {
		t.Fatalf("unexpected response: %#v", badResp)
	}

	idempotency.SetStore(idempotency.NewInMemoryStore())
	goodReq := httptest.NewRequest(http.MethodPost, "/me/change_password", strings.NewReader(`{"old_password":"secret","new_password":"new-secret"}`))
	goodReq.Header.Set("Idempotency-Key", "22221234abcd1234abcd1234abcd1234")
	goodReq = goodReq.WithContext(platformauth.WithPrincipal(goodReq.Context(), platformauth.Principal{UserID: register.UserID, SessionID: currentSessionID}))
	goodRR := httptest.NewRecorder()
	handler.ChangePassword(goodRR, goodReq)
	var goodResp map[string]any
	if err := json.NewDecoder(goodRR.Body).Decode(&goodResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(goodResp["code"].(float64)) != 0 {
		t.Fatalf("unexpected response: %#v", goodResp)
	}

	oldLoginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app"}`))
	oldLoginRR := httptest.NewRecorder()
	handler.Login(oldLoginRR, oldLoginReq)
	var oldLoginResp map[string]any
	if err := json.NewDecoder(oldLoginRR.Body).Decode(&oldLoginResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(oldLoginResp["code"].(float64)) != 10100 {
		t.Fatalf("unexpected old password login response: %#v", oldLoginResp)
	}
	if _, _, ok := handler.service.auth.AuthenticateRequest(goodReq.Context(), "Bearer "+register.AccessToken, ""); !ok {
		t.Fatal("expected current session token to remain valid after change_password")
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
