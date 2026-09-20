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
// nginx 在「链接预览爬虫 UA」分支下反代使用（真实用户走 SPA，见
// docker/nginx.dev.conf / nginx.prod-tls.conf 的 share_crawler map），
// 是 spec「仅 POST RPC」契约的一处登记例外：它是面向爬虫的 HTML 落地页，
// 不属于 /api RPC 面。
//
// 页面只做三件事：og:title/description/image 摘要卡 + 「在 HotSpot 打开」
// 链接（回到同一 URL，由 nginx UA 分支送进 SPA）+ 极简排版。页面为独立
// 落地页，不走前端 design token 体系，内联样式为登记例外。所有动态值
// 经 html.EscapeString 转义（文本与属性同函数，EscapeString 对引号转义
// 后属性安全）。

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
		observability.L().Warn("share og resolve failed",
			"method", "ServeCrawlerHTML", "error", err)
		writeCrawlerPage(w, http.StatusNotFound, "链接已失效",
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
	writeCrawlerPage(w, http.StatusOK, title, description, cover, "/share/"+detail.Share.ShareCode)
}

// writeCrawlerPage 渲染极简 HTML 页。
func writeCrawlerPage(w http.ResponseWriter, status int, title, description, coverURL, openPath string) {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"zh-CN\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + html.EscapeString(title) + " · HotSpot</title>\n")
	b.WriteString("<meta property=\"og:title\" content=\"" + html.EscapeString(title) + "\">\n")
	b.WriteString("<meta property=\"og:description\" content=\"" + html.EscapeString(description) + "\">\n")
	b.WriteString("<meta property=\"og:type\" content=\"website\">\n")
	if coverURL != "" {
		b.WriteString("<meta property=\"og:image\" content=\"" + html.EscapeString(coverURL) + "\">\n")
	}
	b.WriteString("</head>\n")
	b.WriteString("<body style=\"margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f5f7fa;font-family:system-ui,-apple-system,'PingFang SC','Microsoft YaHei',sans-serif;\">\n")
	b.WriteString("<div style=\"max-width:420px;padding:32px;text-align:center;\">\n")
	b.WriteString("<div style=\"font-size:20px;font-weight:600;color:#1a2233;margin-bottom:12px;\">" + html.EscapeString(title) + "</div>\n")
	b.WriteString("<div style=\"font-size:14px;color:#5a6478;line-height:1.6;margin-bottom:24px;\">" + html.EscapeString(description) + "</div>\n")
	b.WriteString("<a href=\"" + html.EscapeString(openPath) + "\" style=\"display:inline-block;padding:10px 28px;border-radius:999px;background:#2e66c0;color:#fff;text-decoration:none;font-size:14px;\">在 HotSpot 中打开</a>\n")
	b.WriteString("</div>\n</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 短缓存：分享摘要随目标内容变化，预览爬虫重取别太勤也不要每次穿透
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(b.String()))
}
