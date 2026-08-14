package auth

import (
	"crypto/rand"
	"net/http"
	"os"
	"time"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
	DeviceIDCookieName     = "device_id"

	// RefreshCookiePath 精确覆盖刷新端点（spec api_spec_base.md「Web 端 Cookie 契约」：
	// refresh_token 仅在 /public/v1/auth/refresh_token 自动携带）。
	RefreshCookiePath = "/public/v1/auth/refresh_token"

	// deviceIDCookieMaxAge = 10 年（长期，独立于登录态；登出不清除）。
	deviceIDCookieMaxAge = 10 * 365 * 24 * time.Hour
)

func isDev() bool {
	return os.Getenv("APP_ENV") == "development"
}

func cookieSecure() bool {
	return !isDev()
}

func cookieSameSite() http.SameSite {
	if isDev() {
		return http.SameSiteLaxMode
	}
	return http.SameSiteStrictMode
}

func SetAuthCookies(w http.ResponseWriter, accessToken string, accessTTL time.Duration, refreshToken string, refreshTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
		MaxAge:   int(accessTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     RefreshCookiePath,
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
		MaxAge:   int(refreshTTL.Seconds()),
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	for _, cookie := range []*http.Cookie{
		{Name: AccessTokenCookieName, Path: "/", HttpOnly: true, Secure: cookieSecure(), SameSite: cookieSameSite(), MaxAge: 0},
		{Name: RefreshTokenCookieName, Path: RefreshCookiePath, HttpOnly: true, Secure: cookieSecure(), SameSite: cookieSameSite(), MaxAge: 0},
	} {
		http.SetCookie(w, cookie)
	}
}

// EnsureDeviceIDCookie implements the Web device-identity cookie contract
// (spec api_spec_base.md): the server issues a fresh UUID v4 via Set-Cookie
// when the bootstrap request arrives without a device_id cookie; an existing
// cookie is kept as-is. Unlike the auth cookies it is NOT HttpOnly (the page
// may read it to share identity with an embedded app WebView) and always
// SameSite=Lax. It survives signout.
func EnsureDeviceIDCookie(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(DeviceIDCookieName); err == nil && c.Value != "" {
		return
	}
	deviceID, err := newUUIDv4()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     DeviceIDCookieName,
		Value:    deviceID,
		Path:     "/",
		HttpOnly: false,
		Secure:   cookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(deviceIDCookieMaxAge.Seconds()),
	})
}

// newUUIDv4 generates a random RFC 4122 version-4 UUID string without
// pulling in an external dependency.
func newUUIDv4() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 0, 36)
	for i, b := range buf {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out = append(out, '-')
		}
		out = append(out, hexDigits[b>>4], hexDigits[b&0x0f])
	}
	return string(out), nil
}
