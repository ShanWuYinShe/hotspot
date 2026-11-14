package share

import (
	"html"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"hotspot/backend-go/internal/platform/observability"
)

// og.go — 分享链接爬虫预览页（GET /share/{code}，挂载见 module.go Mount）。
//
// 背景：前端是 SPA（ssr=false 是架构级禁令，见 frontend/AGENTS.md），外部分享
// 链接被微信/Telegram 等抓取时只能拿到裸 index.html，无 OG 摘要卡。本端点供
// nginx 在「链接预览爬虫 UA 且未带 ?spa=1」分支下反代使用（真实用户走 SPA，
// 见 docker/nginx.dev.conf / nginx.prod-tls.conf 的 $share_og map），
// 是 spec「仅 POST RPC」契约的一处登记例外：它是面向爬虫的 HTML 落地页，
// 不属于 /api RPC 面。
//
// 为何 CTA 要带 ?spa=1：微信/QQ 的内嵌浏览器与它们的预览抓取器共用同一 UA
// 子串（micromessenger / qq/），且 OG 页与 SPA 是同一个 URL——CTA 若指回裸
// URL，内嵌浏览器会再次命中 nginx 的爬虫分支，SPA 分享页永远打不开。带
// ?spa=1 的请求由 nginx 一律送 SPA（含爬虫 UA），形成跳出分支的唯一出口。
//
// 页面只做三件事：og:title/description/url/image 摘要卡 + 「在 HotSpot 打开」
// 链接 + 极简排版。og:url 与 og:image 必须是绝对 URL（微信等预览爬虫通常不
// 解析相对路径），相对 cover 由请求的 scheme+host 补全。响应带
// Vary: User-Agent：同一 URL 按 UA 分流，共享缓存不能把 OG 页发给普通用户。
// 页面为独立落地页，不走前端 design token 体系，内联样式为登记例外。所有
// 动态值经 html.EscapeString 转义（文本与属性同函数，EscapeString 对引号
// 转义后属性安全）。

// shareTypeLabels 分享类型的中文标签（spec api_spec_share.md ShareType）。
var shareTypeLabels = map[string]string{
	"moment": "瞬间",
	"user":   "用户主页",
	"group":  "群组",
}

func (h *Handler) ServeCrawlerHTML(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	detail, err := h.service.Get(r.Context(), GetShareRequest{ShareCode: code})
	if err != nil {
		// 不区分不存在/过期/格式错误——对外一律 404 极简页，不泄露存在性
		observability.FromContext(r.Context()).Warn("share og resolve failed",
			"method", "ServeCrawlerHTML", "error", err)
		writeCrawlerPage(w, r, http.StatusNotFound, "链接已失效",
			"这条分享不存在或已过期", "", "/")
		return
	}

	// 摘要三件套：Preview 由后端在创建分享时生成（spec SharePreview），
	// 缺失时回退创建者昵称 + 类型标签。
	title := "HotSpot 分享"
	if detail.Preview != nil && detail.Preview.Title != "" {
		title = detail.Preview.Title
	}
	description := ""
	if detail.Preview != nil && detail.Preview.Subtitle != nil {
		description = *detail.Preview.Subtitle
	}
	if description == "" {
		if label, ok := shareTypeLabels[detail.Share.ShareType]; ok {
			description = label + " · 来自 " + detail.Share.Creator.Nickname
		}
	}
	cover := ""
	if detail.Preview != nil && detail.Preview.CoverURL != nil {
		cover = *detail.Preview.CoverURL
	}
	// CTA 带 spa=1：微信/QQ 内嵌浏览器与预览抓取器共用 UA 子串，且 OG 页与 SPA
	// 是同一个 URL；不带该参数时 CTA 会再次命中 nginx 的爬虫分支，SPA 分享页
	// 永远打不开。带 spa=1 由 nginx 一律送 SPA（见 docker/nginx*.conf 的 $share_og map）。
	writeCrawlerPage(w, r, http.StatusOK, title, description, cover,
		"/share/"+detail.Share.ShareCode+"?spa=1")
}

// writeCrawlerPage 渲染极简 HTML 页：og:url 取当前页的绝对 URL，相对 cover
// 由请求的 scheme+host 补全（微信等预览爬虫通常不解析相对 og:image）。
func writeCrawlerPage(w http.ResponseWriter, r *http.Request, status int, title, description, coverURL, openPath string) {
	baseURL := requestBaseURL(r)
	canonicalURL := baseURL + r.URL.Path
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + html.EscapeString(title) + " · HotSpot</title>\n")
	b.WriteString("<meta property=\"og:title\" content=\"" + html.EscapeString(title) + "\">\n")
	b.WriteString("<meta property=\"og:description\" content=\"" + html.EscapeString(description) + "\">\n")
	b.WriteString("<meta property=\"og:type\" content=\"website\">\n")
	b.WriteString("<meta property=\"og:url\" content=\"" + html.EscapeString(canonicalURL) + "\">\n")
	if coverURL != "" {
		b.WriteString("<meta property=\"og:image\" content=\"" + html.EscapeString(absoluteURL(baseURL, coverURL)) + "\">\n")
	}
	b.WriteString("</head>\n")
	b.WriteString("<body style=\"margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f5f7fa;font-family:system-ui,-apple-system,'PingFang SC','Microsoft YaHei',sans-serif;\">\n")
	b.WriteString("<div style=\"max-width:420px;padding:32px;text-align:center;\">\n")
	b.WriteString("<div style=\"font-size:20px;font-weight:600;color:#1a2233;margin-bottom:12px;\">" + html.EscapeString(title) + "</div>\n")
	b.WriteString("<div style=\"font-size:14px;color:#5a6478;line-height:1.6;margin-bottom:24px;\">" + html.EscapeString(description) + "</div>\n")
	b.WriteString("<a href=\"" + html.EscapeString(openPath) + "\" style=\"display:inline-block;padding:10px 28px;border-radius:999px;background:#2e66c0;color:#fff;text-decoration:none;font-size:14px;\">在 HotSpot 中打开</a>\n")
	b.WriteString("</div>\n</body>\n</html>\n")

	// 同一 URL 按 UA 分流（爬虫 → OG 页、真实用户 → SPA，见 nginx 的 $share_og
	// map）：共享缓存必须按 User-Agent 分键，否则 OG 页可能被发给普通用户。
	w.Header().Set("Vary", "User-Agent")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 短缓存：分享摘要随目标内容变化，预览爬虫重取别太勤也不要每次穿透
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(b.String()))
}

// requestBaseURL 返回站点绝对前缀 scheme://host。后端不直接对外，scheme 以
// 网关透传的 X-Forwarded-Proto 为准（nginx 用 proxy_set_header 覆盖，客户端
// 无法伪造）；缺失时回退直连 TLS 判断，host 用请求 Host。
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := forwardedProto(r); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// forwardedProto 解析 X-Forwarded-Proto：多级代理可能给出逗号分隔值，取第一
// 个；只接受 http/https，其它取值（含空）视为未提供。
func forwardedProto(r *http.Request) string {
	raw := r.Header.Get("X-Forwarded-Proto")
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = raw[:i]
	}
	proto := strings.ToLower(strings.TrimSpace(raw))
	if proto == "http" || proto == "https" {
		return proto
	}
	return ""
}

// absoluteURL 把站点相对路径补成绝对 URL（微信等预览爬虫通常不解析相对
// og:image）；已是绝对 URL 的原样保留，协议相对路径（//host/path）补上当前
// 请求的 scheme，非站点相对路径（不由后端产生）原样返回。
func absoluteURL(baseURL, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if strings.Contains(target, "://") {
		return target
	}
	if strings.HasPrefix(target, "//") {
		if scheme, _, ok := strings.Cut(baseURL, "://"); ok {
			return scheme + ":" + target
		}
		return target
	}
	if strings.HasPrefix(target, "/") {
		return baseURL + target
	}
	return target
}
