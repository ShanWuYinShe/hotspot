package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// requestOrigin 的优先级与容错：Origin 头优先（浏览器设置、JS 无法伪造），
// "null" 视为无入口信息，Referer 兜底并裁剪成 scheme://host。
func TestMediaRequestOrigin(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"origin header wins", map[string]string{"Origin": " https://chaoset.com ", "Referer": "https://other.example.com/page"}, "https://chaoset.com"},
		{"origin null falls back to referer", map[string]string{"Origin": "null", "Referer": "https://app.example.com/feed?x=1"}, "https://app.example.com"},
		{"referer without path", map[string]string{"Referer": "http://localhost:8888"}, "http://localhost:8888"},
		{"unparsable referer", map[string]string{"Referer": "://bad"}, ""},
		{"no headers", nil, ""},
		{"blank referer", map[string]string{"Referer": "   "}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/media/presign_upload", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}
			if got := requestOrigin(req); got != tc.want {
				t.Fatalf("requestOrigin = %q, want %q", got, tc.want)
			}
		})
	}
}

// ClientEndpointMiddleware 把入口 origin 注入 context：有 scheme 时原样使用；
// 只有 host（"//host:port" 形态的 Origin）时回退 X-Forwarded-Proto，再回退 http；
// 无入口信息时不注入（provider 回退静态 S3_PUBLIC_ENDPOINT）。
func TestMediaClientEndpointMiddleware(t *testing.T) {
	type captured struct {
		endpoint clientRequestEndpoint
		ok       bool
	}
	run := func(t *testing.T, setup func(*http.Request)) captured {
		t.Helper()
		var got captured
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got.endpoint, got.ok = clientRequestEndpointFromContext(r.Context())
			w.WriteHeader(http.StatusNoContent)
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/presign_upload", nil)
		setup(req)
		rr := httptest.NewRecorder()
		ClientEndpointMiddleware(next).ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("next handler not reached, status = %d", rr.Code)
		}
		return got
	}

	withOrigin := run(t, func(r *http.Request) { r.Header.Set("Origin", "https://chaoset.com") })
	if !withOrigin.ok || withOrigin.endpoint != (clientRequestEndpoint{Scheme: "https", Host: "chaoset.com"}) {
		t.Fatalf("origin injection = %#v", withOrigin)
	}

	withForwarded := run(t, func(r *http.Request) {
		r.Header.Set("Origin", "//localhost:8888")
		r.Header.Set("X-Forwarded-Proto", "https")
	})
	if !withForwarded.ok || withForwarded.endpoint != (clientRequestEndpoint{Scheme: "https", Host: "localhost:8888"}) {
		t.Fatalf("forwarded proto injection = %#v", withForwarded)
	}

	defaultScheme := run(t, func(r *http.Request) { r.Header.Set("Origin", "//localhost:8888") })
	if !defaultScheme.ok || defaultScheme.endpoint != (clientRequestEndpoint{Scheme: "http", Host: "localhost:8888"}) {
		t.Fatalf("default scheme injection = %#v", defaultScheme)
	}

	anonymous := run(t, func(*http.Request) {})
	if anonymous.ok {
		t.Fatalf("no origin information must not inject endpoint: %#v", anonymous)
	}
}

func TestWithClientRequestEndpointRequiresHost(t *testing.T) {
	ctx := WithClientRequestEndpoint(context.Background(), "https", "")
	if _, ok := clientRequestEndpointFromContext(ctx); ok {
		t.Fatal("empty host must not inject an endpoint")
	}
	ctx = WithClientRequestEndpoint(ctx, "https", "chaoset.com")
	endpoint, ok := clientRequestEndpointFromContext(ctx)
	if !ok || endpoint.Scheme != "https" || endpoint.Host != "chaoset.com" {
		t.Fatalf("endpoint = %#v ok=%v", endpoint, ok)
	}
}
