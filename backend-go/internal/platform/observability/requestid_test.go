package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddlewareGeneratesHeader(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/public/v1/system/health", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got == "" {
		t.Fatal("expected X-Request-Id header")
	}
}

func TestRequestIDMiddlewarePreservesValidHeader(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/public/v1/system/health", http.NoBody)
	req.Header.Set("X-Request-Id", "1234567890abcdef")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got != "1234567890abcdef" {
		t.Fatalf("unexpected request id: %s", got)
	}
}

// spec api_spec_base.md：客户端携带 X-Request-Id 时服务端原样回显，不施加
// 长度/格式/熵值约束。
func TestRequestIDMiddlewareEchoesShortClientHeader(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != "abc" {
			t.Fatalf("context request id = %q, want abc", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/public/v1/system/health", http.NoBody)
	req.Header.Set("X-Request-Id", "abc")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Request-Id"); got != "abc" {
		t.Fatalf("response request id = %q, want abc", got)
	}
}
