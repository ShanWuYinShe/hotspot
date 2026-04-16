package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/requestmeta"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/resourceguard"
)

func newTestHandler() *Handler {
	cfg := config.Config{AuthTokenSecret: "test-secret"}
	repo := NewInMemoryRepository()
	source := appconfig.NewStaticSource(cfg)
	service := NewServiceWithSource(platformauth.NewService(cfg), repo, cfg, source)
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
	if int(resp["code"].(float64)) != 10000 {
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
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.4.4")
	req.Header.Set("CF-IPCountry", "SG")
	req.Header.Set("User-Agent", "HotSpot App/1.0")
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
	if strings.Count(resp.Data.AccessToken, ".") != 2 {
		t.Fatalf("expected JWT access token, got %q", resp.Data.AccessToken)
	}
}

func TestLoginWebReturnsEmptyDataAndCookiesOnly(t *testing.T) {
	handler := newTestHandler()
	registerAppUser(t, handler, "alice-web", "Alice", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice-web","password":"secret123","client_type":"web"}`))
	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected empty web data, got %#v", resp.Data)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected auth cookies for web login")
	}
}

func TestLoginRejectsUnknownFields(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"alice","password":"secret","client_type":"app","unexpected":true}`))
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if int(resp["code"].(float64)) != 10000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if _, ok := resp["data"]; ok {
		t.Fatalf("expected error response without data, got %#v", resp)
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

func TestListSessionsIncludesSessionMetadata(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	registerReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"login_id":"alice-meta","nickname":"Alice","password":"secret","client_type":"app"}`))
	registerReq.Header.Set("Idempotency-Key", "meta1234abcd1234abcd1234abcd1234")
	registerReq.RemoteAddr = "173.245.48.5:443"
	registerReq.Header.Set("CF-Connecting-IP", "203.0.113.9")
	registerReq.Header.Set("CF-IPCountry", "US")
	registerReq.Header.Set("User-Agent", "HotSpot iOS/1.2.3")
	registerRR := httptest.NewRecorder()
	handler.Register(registerRR, registerReq)
	var registerResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(registerRR.Body).Decode(&registerResp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResp.Code != 0 {
		t.Fatalf("unexpected register response: %#v", registerResp)
	}
	principal, _, ok := handler.service.auth.AuthenticateRequest(context.Background(), "Bearer "+registerResp.Data.AccessToken, "")
	if !ok {
		t.Fatal("expected authenticated principal")
	}

	listReq := httptest.NewRequest(http.MethodPost, "/me/sessions/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(platformauth.WithPrincipal(listReq.Context(), principal))
	listRR := httptest.NewRecorder()
	handler.ListSessions(listRR, listReq)

	var listResp struct {
		Code int         `json:"code"`
		Data SessionList `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Code != 0 || len(listResp.Data.Items) != 1 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
	item := listResp.Data.Items[0]
	if item.DeviceType != "app" || item.DeviceName != "HotSpot iOS/1.2.3" || item.IPAddress != "203.0.xxx.xxx" || item.IPLocation != "US" {
		t.Fatalf("unexpected session item: %#v", item)
	}
}

func TestSessionMetaFromRequestUsesTrustedProxyHeaders(t *testing.T) {
	if err := requestmeta.InitTrustedProxies(""); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "198.51.100.2")
	req.Header.Set("CF-IPCountry", "jp")
	req.Header.Set("User-Agent", "HotSpot Web/9.9")
	meta := sessionMetaFromRequest(req, ClientTypeWeb)
	if meta.DeviceType != "web" || meta.DeviceName != "HotSpot Web/9.9" || meta.IPAddress != "198.51.100.2" || meta.IPLocation != "JP" {
		t.Fatalf("unexpected session meta: %#v", meta)
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
	if int(resp["code"].(float64)) != 10000 {
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

func TestProfileDefaultsUseAppConfigSource(t *testing.T) {
	cfg := config.Config{AuthTokenSecret: "test-secret", DefaultUserAvatar: "legacy-avatar"}
	repo := NewInMemoryRepository()
	source := appconfig.NewStaticSource(cfg)
	_, _ = source.Update(nil, func(current appconfig.AppConfig) appconfig.AppConfig {
		current.Defaults.DefaultUserAvatar = "appconfig-avatar"
		current.Defaults.DefaultLocation.LocationName = "测试位置"
		current.Defaults.DefaultLocation.Point.Latitude = 1.23
		current.Defaults.DefaultLocation.Point.Longitude = 4.56
		return current
	})
	service := NewServiceWithSource(platformauth.NewService(cfg), repo, cfg, source)
	defaults := service.profileDefaults(context.Background())
	if defaults.AvatarURL != "appconfig-avatar" || defaults.LocationName != "测试位置" || defaults.Latitude != 1.23 || defaults.Longitude != 4.56 {
		t.Fatalf("unexpected defaults: %#v", defaults)
	}
}

func TestRegisterRecordsUserForResourceGuard(t *testing.T) {
	checker := &recordingChecker{}
	resourceguard.SetDefaultChecker(checker)
	defer resourceguard.SetDefaultChecker(resourceguard.NoopChecker{})

	repo := &recordingAuthRepo{registerAccount: Account{UserID: "user_new", LoginID: "alice", Nickname: "Alice"}}
	service := NewServiceWithSource(platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"}), repo, config.Config{AuthTokenSecret: "test-secret"}, appconfig.NewStaticSource(config.Config{AuthTokenSecret: "test-secret"}))
	if _, err := service.Register(context.Background(), RegisterRequest{LoginID: "alice", Nickname: "Alice", Password: "secret"}, platformauth.SessionMeta{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(checker.recorded) != 1 || checker.recorded[0] != "user:user_new" {
		t.Fatalf("unexpected records: %#v", checker.recorded)
	}
}

func TestGuestRecordsUserForResourceGuard(t *testing.T) {
	checker := &recordingChecker{}
	resourceguard.SetDefaultChecker(checker)
	defer resourceguard.SetDefaultChecker(resourceguard.NoopChecker{})

	repo := &recordingAuthRepo{guestAccount: Account{UserID: "guest_new", Nickname: "Guest"}}
	service := NewServiceWithSource(platformauth.NewService(config.Config{AuthTokenSecret: "test-secret"}), repo, config.Config{AuthTokenSecret: "test-secret"}, appconfig.NewStaticSource(config.Config{AuthTokenSecret: "test-secret"}))
	if _, err := service.Guest(context.Background(), GuestRequest{Nickname: "Guest"}, platformauth.SessionMeta{}); err != nil {
		t.Fatalf("guest: %v", err)
	}
	if len(checker.recorded) != 1 || checker.recorded[0] != "user:guest_new" {
		t.Fatalf("unexpected records: %#v", checker.recorded)
	}
}

type recordingAuthRepo struct {
	registerAccount Account
	guestAccount    Account
}

type recordingChecker struct {
	recorded []string
}

func (r *recordingAuthRepo) Register(context.Context, RegisterParams) (Account, error) {
	return r.registerAccount, nil
}

func (r *recordingAuthRepo) CreateGuest(context.Context, GuestParams) (Account, error) {
	return r.guestAccount, nil
}

func (r *recordingAuthRepo) Authenticate(context.Context, string, string) (Account, error) {
	return Account{}, ErrInvalidCredentials
}

func (r *recordingAuthRepo) UpdateLoginMeta(context.Context, string, string, string) error {
	return nil
}

func (r *recordingAuthRepo) CreatePasswordReset(context.Context, string, time.Time) (string, error) {
	return "", nil
}

func (r *recordingAuthRepo) ResetPassword(context.Context, string, string, string) (Account, error) {
	return Account{}, ErrPasswordResetNotFound
}

func (r *recordingAuthRepo) ChangePassword(context.Context, string, string, string) error {
	return nil
}

func (c *recordingChecker) Ready(context.Context) bool {
	return true
}

func (c *recordingChecker) MayExist(context.Context, resourceguard.ResourceKind, string) (bool, error) {
	return true, nil
}

func (c *recordingChecker) Record(_ context.Context, kind resourceguard.ResourceKind, id string) error {
	c.recorded = append(c.recorded, string(kind)+":"+id)
	return nil
}
