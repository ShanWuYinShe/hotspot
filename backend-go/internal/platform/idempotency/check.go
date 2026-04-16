package idempotency

import (
	"net/http"
	"regexp"

	"hotspot/backend-go/internal/platform/httpx"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,64}$`)

func RequireKey(r *http.Request) *httpx.APIError {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		return &httpx.APIError{Code: 10000, Message: "缺少 Idempotency-Key"}
	}
	if !keyPattern.MatchString(key) {
		return &httpx.APIError{Code: 22000, Message: "该幂等键格式不合法"}
	}
	*r = *r.WithContext(WithKey(r.Context(), key))
	return nil
}
