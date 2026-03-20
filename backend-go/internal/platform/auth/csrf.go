package auth

import "net/http"

func ValidateCSRF(r *http.Request) bool {
	cookie := readCookie(r, CSRFCookieName)
	header := r.Header.Get(CSRFHeaderName)
	if cookie == "" || header == "" {
		return false
	}
	return cookie == header
}
