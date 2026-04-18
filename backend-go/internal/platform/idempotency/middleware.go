package idempotency

import (
	"net/http"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,64}$`)

// Middleware validates the Idempotency-Key header and injects it into the request context.
// All store checking and response replay is handled by ReplayOrStore inside the handler.
// This avoids duplicating store logic with a mismatched scope key.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" || !keyPattern.MatchString(key) {
			next.ServeHTTP(w, r)
			return
		}
		r = r.WithContext(WithKey(r.Context(), key))
		next.ServeHTTP(w, r)
	})
}
