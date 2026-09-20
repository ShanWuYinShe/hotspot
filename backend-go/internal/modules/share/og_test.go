package share

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

// ogCrawlerRequest 构造带 chi URLParam 的 GET /share/{code} 请求
// （生产由 module.go Mount 的 r.Get("/share/{code}") 注册路由注入）。
func ogCrawlerRequest(code string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/share/"+code, http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestServeCrawlerHTMLRendersOGTags(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
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
}

func TestServeCrawlerHTMLEscapesInterpolation(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
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
}

// TestWriteCrawlerPageEscapesAllInterpolation 直接验证四个插值点
// （标题/描述/封面 URL/打开路径）的 HTML 转义——404 路径全是固定字符串，
// 真正的注入面在 200 页的动态摘要里。
func TestWriteCrawlerPageEscapesAllInterpolation(t *testing.T) {
	rr := httptest.NewRecorder()
	writeCrawlerPage(rr, http.StatusOK,
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
	} {
		if strings.Contains(body, raw) {
			t.Fatalf("unescaped payload leaked into HTML: %s", raw)
		}
	}
	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&#34;") {
		t.Fatalf("expected escaped entities in output: %s", body)
	}
}
