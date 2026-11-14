package share

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/httpx"
)

// ogCrawlerRequest 构造带 chi URLParam 的 GET /share/{code} 请求
// （生产由 module.go Mount 的 r.Get("/share/{code}") 注册路由注入）。
func ogCrawlerRequest(code string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/share/"+code, http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// ogPageRequest 构造带对外 Host 与网关协议头的爬虫请求（生产 nginx 反代时
// Host 透传 $host、X-Forwarded-Proto 透传 $scheme，见 docker/*.conf）。
func ogPageRequest(code, host, proto string) *http.Request {
	req := ogCrawlerRequest(code)
	req.Host = host
	if proto != "" {
		req.Header.Set("X-Forwarded-Proto", proto)
	}
	return req
}

func TestServeCrawlerHTMLRendersOGTags(t *testing.T) {
	handler := NewHandler(NewService())

	// 创建一个 user 分享拿到合法 code
	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create",
		strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq.Header.Set("Idempotency-Key", "ogtestcreatekeyogtestcreatekey0001")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	var createResp struct {
		Code httpx.Code   `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&createResp)
	if createResp.Code != 0 {
		t.Fatalf("create failed: %#v", createResp)
	}

	rr := httptest.NewRecorder()
	handler.ServeCrawlerHTML(rr, ogCrawlerRequest(strings.ToLower(createResp.Data.ShareCode)))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `property="og:title"`) || !strings.Contains(body, `property="og:type"`) {
		t.Fatalf("missing og tags: %s", body)
	}
	if !strings.Contains(body, "在 HotSpot 中打开") {
		t.Fatalf("missing open link: %s", body)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", got)
	}
	// CTA 必须带 spa=1：微信/QQ 内嵌浏览器与预览抓取器同 UA，否 nginx 爬虫分支
	// 会让内嵌浏览器永远停在 OG 页（见 docker/nginx*.conf 的 $share_og map）。
	if !strings.Contains(body, "?spa=1") {
		t.Fatalf("missing spa=1 escape-hatch in CTA: %s", body)
	}
	if got := rr.Header().Get("Vary"); got != "User-Agent" {
		t.Fatalf("expected Vary: User-Agent, got %q", got)
	}
}

func TestServeCrawlerHTMLEscapesInterpolation(t *testing.T) {
	handler := NewHandler(NewService())
	// 不存在的 code 走 404 页（固定文案，攻击面为零）；恶意形态输入不得原样出现
	rr := httptest.NewRecorder()
	handler.ServeCrawlerHTML(rr, ogCrawlerRequest(`"><script>alert(1)</script>`))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown code, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "<script>alert(1)</script>") {
		t.Fatalf("unescaped payload leaked into HTML: %s", rr.Body.String())
	}
	// 404 页同样带 Vary：UA 分流对命中/未命中路径都成立
	if got := rr.Header().Get("Vary"); got != "User-Agent" {
		t.Fatalf("expected Vary: User-Agent on 404 page, got %q", got)
	}
}

// TestWriteCrawlerPageEscapesAllInterpolation 直接验证五个插值点
// （标题/描述/封面 URL/打开路径/og:url 中的 Host）的 HTML 转义——404 路径
// 全是固定字符串，真正的注入面在 200 页的动态摘要里。
func TestWriteCrawlerPageEscapesAllInterpolation(t *testing.T) {
	rr := httptest.NewRecorder()
	writeCrawlerPage(rr, ogPageRequest("ABCDEFGHJKLM", `example.com"><script>alert(4)</script>`, "https"), http.StatusOK,
		`<script>alert("title")</script>`,
		`desc "></script><img src=x onerror=alert(1)>`,
		`http://x/cover.png"><script>alert(2)</script>`,
		`/share/a"onmouseover="alert(3)`)
	body := rr.Body.String()
	for _, raw := range []string{
		`<script>alert("title")</script>`,
		`"></script><img src=x onerror=alert(1)>`,
		`"><script>alert(2)</script>`,
		`"onmouseover="alert(3)`,
		`<script>alert(4)</script>`,
	} {
		if strings.Contains(body, raw) {
			t.Fatalf("unescaped payload leaked into HTML: %s", raw)
		}
	}
	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&#34;") {
		t.Fatalf("expected escaped entities in output: %s", body)
	}
}

// TestServeCrawlerHTMLAbsoluteURLs 锁定对外 URL 契约：og:url 是当前页绝对
// URL，相对 cover 用请求的 scheme+host 绝对化（微信等预览爬虫不解析相对路径）。
func TestServeCrawlerHTMLAbsoluteURLs(t *testing.T) {
	cover := "/public/v1/media/avatars/file_abc"
	subtitle := "副标题"
	handler := NewHandler(NewServiceWithRepo(&shareStaticGetRepo{
		shareStubRepo: &shareStubRepo{},
		detail: ShareDetail{
			Share:   ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
			Preview: &SharePreview{Title: "标题", Subtitle: &subtitle, CoverURL: &cover},
		},
	}))

	rr := httptest.NewRecorder()
	handler.ServeCrawlerHTML(rr, ogPageRequest("ABCDEFGHJKLM", "hotspot.example.com", "https"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`<meta property="og:url" content="https://hotspot.example.com/share/ABCDEFGHJKLM">`,
		`<meta property="og:image" content="https://hotspot.example.com/public/v1/media/avatars/file_abc">`,
		`href="/share/ABCDEFGHJKLM?spa=1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in crawler page: %s", want, body)
		}
	}
}

// TestServeCrawlerHTMLKeepsAbsoluteCoverURL 已是绝对 URL 的 cover 原样输出；
// 协议相对（//host/path）补当前请求 scheme，不重复拼 host。
func TestServeCrawlerHTMLKeepsAbsoluteCoverURL(t *testing.T) {
	cases := []struct {
		name  string
		cover string
		want  string
	}{
		{"absolute", "https://cdn.example.com/cover.png", "https://cdn.example.com/cover.png"},
		{"protocol relative", "//cdn.example.com/cover.png", "https://cdn.example.com/cover.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cover := tc.cover
			handler := NewHandler(NewServiceWithRepo(&shareStaticGetRepo{
				shareStubRepo: &shareStubRepo{},
				detail: ShareDetail{
					Share:   ShareBase{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, Creator: ShareUserInfo{UserID: "user_bob", Nickname: "Bob"}},
					Preview: &SharePreview{Title: "标题", CoverURL: &cover},
				},
			}))
			rr := httptest.NewRecorder()
			handler.ServeCrawlerHTML(rr, ogPageRequest("ABCDEFGHJKLM", "hotspot.example.com", "https"))
			if !strings.Contains(rr.Body.String(), `<meta property="og:image" content="`+tc.want+`">`) {
				t.Fatalf("cover %q not emitted as %q: %s", tc.cover, tc.want, rr.Body.String())
			}
		})
	}
}

// TestRequestBaseURL 覆盖 scheme 推导分支：X-Forwarded-Proto 优先且只接受
// http/https（多级代理逗号值取第一个），缺失时回退直连 TLS 判断。
func TestRequestBaseURL(t *testing.T) {
	cases := []struct {
		name  string
		proto string
		tls   bool
		want  string
	}{
		{"forwarded https", "https", false, "https://hotspot.example.com"},
		{"forwarded http", "http", false, "http://hotspot.example.com"},
		{"forwarded list takes first", "https, http", false, "https://hotspot.example.com"},
		{"invalid proto falls back to plain http", "javascript", false, "http://hotspot.example.com"},
		{"direct tls", "", true, "https://hotspot.example.com"},
		{"plain http", "", false, "http://hotspot.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ogPageRequest("ABCDEFGHJKLM", "hotspot.example.com", tc.proto)
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if got := requestBaseURL(req); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

// TestAbsoluteURL 覆盖 og:image 绝对化分支（站点相对/绝对/协议相对/空/普通相对）。
func TestAbsoluteURL(t *testing.T) {
	const base = "https://hotspot.example.com"
	cases := []struct {
		name   string
		base   string
		target string
		want   string
	}{
		{"empty", base, "", ""},
		{"site relative", base, "/public/v1/media/avatars/file_abc", base + "/public/v1/media/avatars/file_abc"},
		{"absolute unchanged", base, "https://cdn.example.com/cover.png", "https://cdn.example.com/cover.png"},
		{"protocol relative gets scheme", base, "//cdn.example.com/cover.png", "https://cdn.example.com/cover.png"},
		{"protocol relative without base scheme", "", "//cdn.example.com/cover.png", "//cdn.example.com/cover.png"},
		{"plain relative unchanged", base, "uploads/cover.png", "uploads/cover.png"},
		{"trim spaces", base, "  /cover.png  ", base + "/cover.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := absoluteURL(tc.base, tc.target); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
