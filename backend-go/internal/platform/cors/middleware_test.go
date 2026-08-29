package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(cfg Config, origin, method string) *httptest.ResponseRecorder {
	handler := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// 白名单 Origin 精确匹配：允许项回显 Origin，未列入项不写 CORS 头。
func TestOriginWhitelistExactMatch(t *testing.T) {
	cfg := Config{AllowedOrigins: []string{"https://a.example", "http://b.example"}, AllowedMethods: []string{"GET"}}
	allowed := serve(cfg, "https://a.example", http.MethodGet)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if allowed.Header().Get("Vary") != "Origin" {
		t.Fatal("expected Vary: Origin for whitelisted origin")
	}
	denied := serve(cfg, "https://evil.example", http.MethodGet)
	if got := denied.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unlisted origin must not get CORS headers, got %q", got)
	}
}

// 通配 "*"：直接回 * 且不带 Vary；无 Origin 请求直通。
func TestWildcardAndNoOrigin(t *testing.T) {
	cfg := DefaultConfig()
	wild := serve(cfg, "https://any.example", http.MethodGet)
	if got := wild.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected *, got %q", got)
	}
	noOrigin := serve(cfg, "", http.MethodGet)
	if noOrigin.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("request without Origin should pass through untouched")
	}
}

// 预检（OPTIONS）：204 + Methods/Headers/MaxAge；普通请求透传下一处理器。
func TestPreflightHeaders(t *testing.T) {
	cfg := DefaultConfig()
	preflight := serve(cfg, "https://a.example", http.MethodOptions)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", preflight.Code)
	}
	if preflight.Header().Get("Access-Control-Allow-Methods") == "" || preflight.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("preflight must list allow methods/headers")
	}
	if preflight.Header().Get("Access-Control-Max-Age") != "300" {
		t.Fatalf("Max-Age = %q, want 300", preflight.Header().Get("Access-Control-Max-Age"))
	}
	normal := serve(cfg, "https://a.example", http.MethodGet)
	if normal.Code != http.StatusOK {
		t.Fatalf("normal request status = %d, want 200", normal.Code)
	}
}

// AllowCredentials 与 ExposedHeaders 透出。
func TestCredentialsAndExposedHeaders(t *testing.T) {
	cfg := Config{
		AllowedOrigins:   []string{"https://a.example"},
		AllowCredentials: true,
		ExposedHeaders:   []string{"X-Request-ID"},
	}
	res := serve(cfg, "https://a.example", http.MethodGet)
	if res.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected Allow-Credentials: true")
	}
	if res.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" {
		t.Fatal("expected exposed headers")
	}
}
