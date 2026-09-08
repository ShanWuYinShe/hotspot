package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestNewUUIDv4Format(t *testing.T) {
	// RFC 4122 v4: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for i := 0; i < 20; i++ {
		id, err := newUUIDv4()
		if err != nil {
			t.Fatalf("newUUIDv4 failed: %v", err)
		}
		if !uuidRegex.MatchString(id) {
			t.Fatalf("uuid %q does not match RFC 4122 v4 spec", id)
		}
	}
}

func TestSetAndClearAuthCookiesPolicy(t *testing.T) {
	// 测试生产模式（非 development）
	_ = os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	rr := httptest.NewRecorder()
	SetAuthCookies(rr, "test_access_token", 15*time.Minute, "test_refresh_token", 7*24*time.Hour)

	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	var accessCookie, refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == AccessTokenCookieName {
			accessCookie = c
		} else if c.Name == RefreshTokenCookieName {
			refreshCookie = c
		}
	}

	if accessCookie == nil || refreshCookie == nil {
		t.Fatal("expected both access and refresh cookies")
	}

	// 验证 accessCookie
	if accessCookie.Value != "test_access_token" || accessCookie.Path != "/" || !accessCookie.HttpOnly || !accessCookie.Secure || accessCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected accessCookie config: %+v", accessCookie)
	}

	// 验证 refreshCookie 必须严格限制在 RefreshCookiePath
	if refreshCookie.Value != "test_refresh_token" || refreshCookie.Path != RefreshCookiePath || !refreshCookie.HttpOnly || !refreshCookie.Secure {
		t.Fatalf("unexpected refreshCookie config: %+v", refreshCookie)
	}

	// 测试 ClearAuthCookies
	clearRR := httptest.NewRecorder()
	ClearAuthCookies(clearRR)
	clearCookies := clearRR.Result().Cookies()
	if len(clearCookies) != 2 {
		t.Fatalf("expected 2 clear cookies, got %d", len(clearCookies))
	}
	for _, c := range clearCookies {
		if c.MaxAge != 0 {
			t.Fatalf("expected MaxAge 0 for cleared cookie %s, got %d", c.Name, c.MaxAge)
		}
	}
}

func TestEnsureDeviceIDCookie(t *testing.T) {
	// 1. 无 Cookie 时应下发新鲜的 device_id，且非 HttpOnly，SameSite=Lax
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	EnsureDeviceIDCookie(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie set, got %d", len(cookies))
	}
	devCookie := cookies[0]
	if devCookie.Name != DeviceIDCookieName || devCookie.Value == "" {
		t.Fatalf("expected valid device_id cookie, got %+v", devCookie)
	}
	if devCookie.HttpOnly {
		t.Fatal("device_id cookie must NOT be HttpOnly so client page can read it")
	}
	if devCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax, got %v", devCookie.SameSite)
	}

	// 2. 已有 Cookie 时不应重复下发
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(&http.Cookie{Name: DeviceIDCookieName, Value: "existing-device-uuid"})
	EnsureDeviceIDCookie(rr2, req2)

	if len(rr2.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookie set when device_id already exists, got %+v", rr2.Result().Cookies())
	}
}

func TestDevEnvironmentCookieSettings(t *testing.T) {
	_ = os.Setenv("APP_ENV", "development")
	defer os.Unsetenv("APP_ENV")

	if cookieSecure() {
		t.Fatal("expected insecure cookies in development mode")
	}
	if cookieSameSite() != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax in dev mode, got %v", cookieSameSite())
	}
}
