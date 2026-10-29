package media

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// ── 客户端可见端点的入口 origin 注入 ────────────────────────────────
// 上传直传与媒体读 URL 必须与用户访问站点的 origin 同源：SigV4 签名含 Host
// （nginx /s3/ 以 $http_host 原样转发），且浏览器跨源 XHR/img 会被 CORS 与
// Local Network Access 拦截（静态 S3_PUBLIC_ENDPOINT 如 127.0.0.1:8888，在
// 用户从 localhost/域名/https 入口访问时全部命中）。入口 origin 从请求的
// Origin/Referer 头提取（天然含端口），经 ClientEndpointMiddleware 注入
// context；无入口信息（脚本/内部调用）时回退静态 S3_PUBLIC_ENDPOINT。

type clientRequestEndpointKey struct{}

type clientRequestEndpoint struct {
	Scheme string
	Host   string
}

// WithClientRequestEndpoint 把请求入口 origin 注入 context（host 为空视为无入口信息）。
func WithClientRequestEndpoint(ctx context.Context, scheme, host string) context.Context {
	if host == "" {
		return ctx
	}
	return context.WithValue(ctx, clientRequestEndpointKey{}, clientRequestEndpoint{Scheme: scheme, Host: host})
}

func clientRequestEndpointFromContext(ctx context.Context) (clientRequestEndpoint, bool) {
	ep, ok := ctx.Value(clientRequestEndpointKey{}).(clientRequestEndpoint)
	return ep, ok && ep.Host != ""
}

// requestOrigin 返回发起请求的页面 origin（Origin 头优先，Referer 兜底），
// 形如 http://localhost:8888。
func requestOrigin(r *http.Request) string {
	if o := strings.TrimSpace(r.Header.Get("Origin")); o != "" && o != "null" {
		return o
	}
	if ref := strings.TrimSpace(r.Header.Get("Referer")); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	return ""
}

// ClientEndpointMiddleware 为所有 API 请求注入入口 origin（供媒体模块生成
// 同源的直传/读 URL）。scheme/host 都取 Origin/Referer（含端口）：这是浏览器
// 实际访问的 origin，JS 无法伪造（Origin 由浏览器设置、Referer 受浏览器控制），
// 也天然覆盖「TLS 在网关/隧道终止、nginx 侧 $scheme 已是 http」的部署形态
// （OrbStack .orb.local HTTPS 实测：X-Forwarded-Proto=http，若以其为准会签发
// http 图片 URL，https 页面被 CSP img-src 拦截 + 混合内容阻断）。仅当无
// Origin/Referer（脚本/内部调用）时才回退 X-Forwarded-Proto。
func ClientEndpointMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ep := requestOrigin(r); ep != "" {
			if u, err := url.Parse(ep); err == nil && u.Host != "" {
				scheme := u.Scheme
				if scheme == "" {
					scheme = "http"
					if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
						scheme = proto
					}
				}
				r = r.WithContext(WithClientRequestEndpoint(r.Context(), scheme, u.Host))
			}
		}
		next.ServeHTTP(w, r)
	})
}
