package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/testdb"
)

// newCovHandlerWithRepo 用给定 Repository 装配 handler（用于覆盖 service 错误分支）。
func newCovHandlerWithRepo(repo Repository) *Handler {
	cfg := config.Config{AuthTokenSecret: "test-secret"}
	svc := NewServiceWithSource(context.Background(), platformauth.NewService(cfg), repo, cfg, appconfig.NewStaticSource(cfg))
	return NewHandler(svc)
}

func covUniqueKey(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%032x", time.Now().UnixNano())
}

func TestHandlerRegisterValidationBranchesCov(t *testing.T) {
	handler := newTestHandler()
	cases := []struct{ name, body string }{
		{"malformed json", `{invalid`},
		{"bad client type", `{"login_id":"alice","nickname":"A","password":"password1","client_type":"cli","device_id":"d1"}`},
		{"bad password", `{"login_id":"alice","nickname":"A","password":"short","client_type":"app","device_id":"d1"}`},
		{"missing device", `{"login_id":"alice","nickname":"A","password":"password1","client_type":"app"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.body))
			req.Header.Set("Idempotency-Key", covUniqueKey(t))
			rr := httptest.NewRecorder()
			handler.Register(rr, req)
			if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
				t.Fatalf("%s: expected 10000, got %d", tc.name, code)
			}
		})
	}
}

func TestHandlerRegisterWebServiceErrorCov(t *testing.T) {
	handler := newTestHandler()
	loginID := fmt.Sprintf("cov_web_dup_%d", time.Now().UnixNano())
	body := func() string {
		return `{"login_id":"` + loginID + `","nickname":"Web","password":"password1","client_type":"web","device_id":"cov-web-device"}`
	}
	first := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body()))
	first.Header.Set("Idempotency-Key", covUniqueKey(t))
	firstRR := httptest.NewRecorder()
	handler.Register(firstRR, first)
	if code := testdb.DecodeAPICode(t, firstRR); code != 0 {
		t.Fatalf("first web register: expected success, got %d", code)
	}

	second := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body()))
	second.Header.Set("Idempotency-Key", covUniqueKey(t))
	secondRR := httptest.NewRecorder()
	handler.Register(secondRR, second)
	if code := testdb.DecodeAPICode(t, secondRR); code != httpx.CodeConflict {
		t.Fatalf("duplicate web register: expected 10400, got %d", code)
	}
}

func TestHandlerLoginMissingDeviceIDCov(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"cov_user","password":"password1","client_type":"app"}`))
	rr := httptest.NewRecorder()
	handler.Login(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for missing device_id, got %d", code)
	}
}

func TestHandlerGuestWebAndValidationBranchesCov(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"nickname":"WebGuest","client_type":"web","device_id":"cov-guest-web"}`))
	req.Header.Set("Idempotency-Key", covUniqueKey(t))
	rr := httptest.NewRecorder()
	handler.Guest(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != 0 {
		t.Fatalf("web guest: expected success, got %d", code)
	}
	foundRefresh := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == platformauth.RefreshTokenCookieName {
			foundRefresh = true
		}
	}
	if !foundRefresh {
		t.Fatal("web guest must set refresh_token cookie")
	}

	for _, body := range []string{
		`{"nickname":"Guest","client_type":"cli","device_id":"d1"}`,
		`{"nickname":"Guest","client_type":"app"}`,
	} {
		badReq := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(body))
		badReq.Header.Set("Idempotency-Key", covUniqueKey(t))
		badRR := httptest.NewRecorder()
		handler.Guest(badRR, badReq)
		if code := testdb.DecodeAPICode(t, badRR); code != httpx.CodeInvalidRequest {
			t.Fatalf("body %s: expected 10000, got %d", body, code)
		}
	}
}

func TestHandlerGuestWebServiceErrorCov(t *testing.T) {
	handler := newCovHandlerWithRepo(&stubAuthRepo{guestErr: errors.New("repo down")})
	req := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"nickname":"WebGuest","client_type":"web","device_id":"cov-guest-web"}`))
	req.Header.Set("Idempotency-Key", covUniqueKey(t))
	rr := httptest.NewRecorder()
	handler.Guest(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600, got %d", code)
	}
}

func TestHandlerRefreshTokenBranchesCov(t *testing.T) {
	handler := newTestHandler()

	malformed := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(`{invalid`))
	malformedRR := httptest.NewRecorder()
	handler.RefreshToken(malformedRR, malformed)
	if code := testdb.DecodeAPICode(t, malformedRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("malformed: expected 10000, got %d", code)
	}

	noDevice := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(`{"client_type":"app","refresh_token":"tok"}`))
	noDeviceRR := httptest.NewRecorder()
	handler.RefreshToken(noDeviceRR, noDevice)
	if code := testdb.DecodeAPICode(t, noDeviceRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("missing device: expected 10000, got %d", code)
	}

	invalid := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(`{"client_type":"app","device_id":"d1","refresh_token":"invalid-token"}`))
	invalidRR := httptest.NewRecorder()
	handler.RefreshToken(invalidRR, invalid)
	if code := testdb.DecodeAPICode(t, invalidRR); code != httpx.CodeUnauthorized {
		t.Fatalf("invalid token: expected 10100, got %d", code)
	}

	loginID := fmt.Sprintf("cov_refresh_web_%d", time.Now().UnixNano())
	regReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(
		`{"login_id":"`+loginID+`","nickname":"Web","password":"password1","client_type":"web","device_id":"cov-refresh-device"}`))
	regReq.Header.Set("Idempotency-Key", covUniqueKey(t))
	regRR := httptest.NewRecorder()
	handler.Register(regRR, regReq)
	if code := testdb.DecodeAPICode(t, regRR); code != 0 {
		t.Fatalf("web register: expected success, got %d", code)
	}
	refreshReq := httptest.NewRequest(http.MethodPost, "/refresh_token", strings.NewReader(`{"client_type":"web","device_id":"cov-refresh-device"}`))
	for _, c := range regRR.Result().Cookies() {
		refreshReq.AddCookie(c)
	}
	refreshRR := httptest.NewRecorder()
	handler.RefreshToken(refreshRR, refreshReq)
	if code := testdb.DecodeAPICode(t, refreshRR); code != 0 {
		t.Fatalf("web refresh: expected success, got %d", code)
	}
	rotated := false
	for _, c := range refreshRR.Result().Cookies() {
		if c.Name == platformauth.RefreshTokenCookieName && c.Value != "" {
			rotated = true
		}
	}
	if !rotated {
		t.Fatal("web refresh must rotate refresh_token cookie")
	}
}

func TestHandlerSendRegisterCodeServiceErrorBranchesCov(t *testing.T) {
	handler := newCovHandlerWithRepo(&stubAuthRepo{registerCodeErr: errors.New("repo down")})
	req := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(`{"login_id":"cov_send_err@example.com","client_type":"app"}`))
	rr := httptest.NewRecorder()
	handler.SendRegisterCode(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600, got %d", code)
	}

	throttled := newCovHandlerWithRepo(&stubAuthRepo{registerCodeErr: ErrRegisterCodeThrottled})
	throttleReq := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(`{"login_id":"cov_send_throttle@example.com","client_type":"app"}`))
	throttleRR := httptest.NewRecorder()
	throttled.SendRegisterCode(throttleRR, throttleReq)
	if code := testdb.DecodeAPICode(t, throttleRR); code != httpx.CodeRateLimited {
		t.Fatalf("expected 10500, got %d", code)
	}
	if throttleRR.Header().Get("Retry-After") == "" {
		t.Fatal("throttled send must carry Retry-After")
	}

	malformed := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(`{invalid`))
	malformedRR := httptest.NewRecorder()
	throttled.SendRegisterCode(malformedRR, malformed)
	if code := testdb.DecodeAPICode(t, malformedRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("malformed: expected 10000, got %d", code)
	}
}

func TestHandlerSendRegisterCodeThrottledPerAccountCov(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	if _, err := source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.RateLimits.AuthRegisterPer10Minutes = 1
		return cfg
	}); err != nil {
		t.Fatalf("configure source: %v", err)
	}
	ratelimit.SetSource(source)
	t.Cleanup(func() { ratelimit.SetSource(appconfig.NewStaticSource(config.Config{})) })

	handler := newTestHandler()
	handler.service.codeSender = &logVerificationSender{}
	loginID := fmt.Sprintf("cov_send_throttle_%d@example.com", time.Now().UnixNano())
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(
			`{"login_id":"`+loginID+`","client_type":"app"}`))
		rr := httptest.NewRecorder()
		handler.SendRegisterCode(rr, req)
		return rr
	}
	if code := testdb.DecodeAPICode(t, send()); code != 0 {
		t.Fatalf("first send should pass, got %d", code)
	}
	limited := send()
	if code := testdb.DecodeAPICode(t, limited); code != httpx.CodeRateLimited {
		t.Fatalf("second send should be throttled, got %d", code)
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("throttled send must carry Retry-After")
	}
}

func TestHandlerRequestPasswordResetMalformedBodyCov(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(`{invalid`))
	rr := httptest.NewRecorder()
	handler.RequestPasswordReset(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000, got %d", code)
	}
}

func TestHandlerResetPasswordBranchesCov(t *testing.T) {
	handler := newTestHandler()
	handler.service.codeSender = &logVerificationSender{}

	malformed := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(`{invalid`))
	malformedRR := httptest.NewRecorder()
	handler.ResetPassword(malformedRR, malformed)
	if code := testdb.DecodeAPICode(t, malformedRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("malformed: expected 10000, got %d", code)
	}

	badPassword := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(
		`{"login_id":"a@b.com","verification_code":"123456","new_password":"short","client_type":"app","device_id":"d1"}`))
	badPasswordRR := httptest.NewRecorder()
	handler.ResetPassword(badPasswordRR, badPassword)
	if code := testdb.DecodeAPICode(t, badPasswordRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("bad password: expected 10000, got %d", code)
	}

	noDevice := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(
		`{"login_id":"a@b.com","verification_code":"123456","new_password":"password1","client_type":"app"}`))
	noDeviceRR := httptest.NewRecorder()
	handler.ResetPassword(noDeviceRR, noDevice)
	if code := testdb.DecodeAPICode(t, noDeviceRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("missing device: expected 10000, got %d", code)
	}

	webBad := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(
		`{"login_id":"cov_missing_reset@example.com","verification_code":"123456","new_password":"password1","client_type":"web","device_id":"cov-reset-web"}`))
	webBadRR := httptest.NewRecorder()
	handler.ResetPassword(webBadRR, webBad)
	if code := testdb.DecodeAPICode(t, webBadRR); code != httpx.CodeNotFound {
		t.Fatalf("web bad code: expected 404, got %d", code)
	}

	loginID := fmt.Sprintf("cov_reset_web_%d", time.Now().UnixNano())
	regReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(
		`{"login_id":"`+loginID+`","nickname":"WebReset","password":"password1","client_type":"app","device_id":"cov-reset-device"}`))
	regReq.Header.Set("Idempotency-Key", covUniqueKey(t))
	regRR := httptest.NewRecorder()
	handler.Register(regRR, regReq)
	if code := testdb.DecodeAPICode(t, regRR); code != 0 {
		t.Fatalf("register: expected success, got %d", code)
	}
	resetReq := httptest.NewRequest(http.MethodPost, "/request_password_reset", strings.NewReader(
		`{"login_id":"`+loginID+`","client_type":"app"}`))
	resetRR := httptest.NewRecorder()
	handler.RequestPasswordReset(resetRR, resetReq)
	if code := testdb.DecodeAPICode(t, resetRR); code != 0 {
		t.Fatalf("request reset: expected success, got %d", code)
	}
	code := handler.service.repo.(*InMemoryRepository).LastResetCode(loginID)
	if code == "" {
		t.Fatal("expected reset code")
	}
	confirmReq := httptest.NewRequest(http.MethodPost, "/reset_password", strings.NewReader(
		`{"login_id":"`+loginID+`","verification_code":"`+code+`","new_password":"newpassword1","client_type":"web","device_id":"cov-reset-device"}`))
	confirmRR := httptest.NewRecorder()
	handler.ResetPassword(confirmRR, confirmReq)
	if got := testdb.DecodeAPICode(t, confirmRR); got != 0 {
		t.Fatalf("web reset: expected success, got %d", got)
	}
}

func TestHandlerChangePasswordValidationBranchesCov(t *testing.T) {
	handler := newTestHandler()
	cases := []string{
		`{invalid`,
		`{"old_password":"","new_password":"password1"}`,
		`{"old_password":"old","new_password":"short"}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/change_password", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.ChangePassword(rr, req)
		if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
			t.Fatalf("body %q: expected 10000, got %d", body, code)
		}
	}
}

func TestHandlerListSessionsAndLogoutRejectNonObjectCov(t *testing.T) {
	handler := newTestHandler()
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"list_sessions": handler.ListSessions,
		"logout":        handler.Logout,
	} {
		req := httptest.NewRequest(http.MethodPost, "/"+name, strings.NewReader(`[1]`))
		rr := httptest.NewRecorder()
		call(rr, req)
		if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
			t.Fatalf("%s: expected 10000, got %d", name, code)
		}
	}
}

func TestErrorResponseNilFallbackCov(t *testing.T) {
	resp := errorResponse[AuthTokenPairApp](nil)
	if resp.Code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600 fallback, got %d", resp.Code)
	}
}

func TestSessionMetaTruncatesLongUserAgentCov(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req.Header.Set("User-Agent", strings.Repeat("u", 200))
	meta := sessionMetaFromRequest(req, ClientTypeApp, "d1")
	if len(meta.DeviceName) != 120 {
		t.Fatalf("expected UA truncated to 120 bytes, got %d", len(meta.DeviceName))
	}
}

func TestResolveDeviceIDBranchesCov(t *testing.T) {
	body := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	body.AddCookie(&http.Cookie{Name: platformauth.DeviceIDCookieName, Value: "cookie-device"})
	if got := resolveDeviceID(body, "body-device"); got != "body-device" {
		t.Fatalf("body device should win, got %q", got)
	}
	if got := resolveDeviceID(body, ""); got != "cookie-device" {
		t.Fatalf("cookie device should be used, got %q", got)
	}
	empty := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	if got := resolveDeviceID(empty, ""); got != "" {
		t.Fatalf("expected empty device, got %q", got)
	}
}
