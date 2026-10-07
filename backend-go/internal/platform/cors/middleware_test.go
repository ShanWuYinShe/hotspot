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
// 默认配置（同源部署）：跨源一律不放行——不发 ACAO，请求原样透传；
// 无 Origin 的同源请求同样透传。白名单经 WithAllowedOrigins 显式开启。
func TestDefaultConfigRejectsCrossOrigin(t *testing.T) {
	cfg := DefaultConfig()
	cross := serve(cfg, "https://any.example", http.MethodGet)
	if cross.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("default config must not emit ACAO for cross-origin requests")
	}
	if cross.Code != http.StatusOK {
		t.Fatalf("rejected cross-origin request should pass through to handler, got %d", cross.Code)
	}
	noOrigin := serve(cfg, "", http.MethodGet)
	if noOrigin.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("request without Origin should pass through untouched")
	}
}

func TestAllowedOriginsWhitelist(t *testing.T) {
	cfg := DefaultConfig().WithAllowedOrigins([]string{"https://a.example"})
	allowed := serve(cfg, "https://a.example", http.MethodGet)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example" {
		t.Fatalf("expected origin echo, got %q", got)
	}
	denied := serve(cfg, "https://evil.example", http.MethodGet)
	if denied.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("non-whitelisted origin must not receive ACAO")
	}
}

// 预检（OPTIONS）：白名单内 204 + Methods/Headers/MaxAge；普通请求透传
// 下一处理器；白名单外预检同样透传（无 CORS 头）。
func TestPreflightHeaders(t *testing.T) {
	cfg := DefaultConfig().WithAllowedOrigins([]string{"https://a.example"})
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

// 通配 "*" + AllowCredentials：失败安全——不发放凭证头。
// Fetch 规范本身禁止该组合（浏览器会拒绝响应），且它等价于「任意站点带凭证跨源」；
// 显式白名单下凭证照常发放（见 TestCredentialsAndExposedHeaders）。
func TestWildcardDoesNotEmitCredentials(t *testing.T) {
	wildcard := serve(Config{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowCredentials: true,
	}, "https://any.example", http.MethodGet)
	if got := wildcard.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("wildcard origin should still be echoed as *, got %q", got)
	}
	if got := wildcard.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard + credentials must not emit credentials header, got %q", got)
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
