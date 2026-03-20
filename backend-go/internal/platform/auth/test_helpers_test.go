package auth

import (
	"net/http"
	"net/http/httptest"
)

func newRequestWithCookie(name, value string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: name, Value: value})
	return req
}
