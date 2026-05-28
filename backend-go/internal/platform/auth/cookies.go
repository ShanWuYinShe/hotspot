package auth

import (
	"net/http"
	"os"
	"time"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
	CSRFCookieName         = "hs_csrf_token"
	CSRFHeaderName         = "X-CSRF-Token"
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

func SetAuthCookies(w http.ResponseWriter, accessToken string, accessTTL time.Duration, refreshToken string, refreshTTL time.Duration, csrfToken string) {
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
		Path:     "/api/v1/auth/refresh_token",
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
		MaxAge:   int(refreshTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   cookieSecure(),
		SameSite: cookieSameSite(),
		MaxAge:   int(accessTTL.Seconds()),
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	for _, cookie := range []*http.Cookie{
		{Name: AccessTokenCookieName, Path: "/", HttpOnly: true, Secure: cookieSecure(), SameSite: cookieSameSite(), MaxAge: 0},
		{Name: RefreshTokenCookieName, Path: "/api/v1/auth/refresh_token", HttpOnly: true, Secure: cookieSecure(), SameSite: cookieSameSite(), MaxAge: 0},
		{Name: CSRFCookieName, Path: "/", HttpOnly: false, Secure: cookieSecure(), SameSite: cookieSameSite(), MaxAge: 0},
	} {
		http.SetCookie(w, cookie)
	}
}
