# ── HotSpot 生产 server{} 公共片段（80/443 共用）──────────────────
# 位置类配置与仓库 docker/nginx.conf 保持一致；依赖同目录挂载的
# headers.inc / proxy.inc / amap.inc。

root /usr/share/nginx/html;
client_max_body_size 100m;
server_tokens off;

# ── 入口访问控制：仅 Cloudflare 回源网段与容器内本机 ──────────────
# 本站仅经 CF full proxy 对外服务，源站 IPv6 全网可路由，不锁定即可被
# 直连绕过 CF 的 WAF/边缘限流。80/443 双块共用本 include，一并生效。
# 网段来源：https://www.cloudflare.com/ips-v6（2026-09 核对）
# 127.0.0.1/::1 供容器 healthcheck（wget http://127.0.0.1/）与本地排查。
allow 127.0.0.1;
allow ::1;
allow 2400:cb00::/32;
allow 2606:4700::/32;
allow 2803:f800::/32;
allow 2405:b500::/32;
allow 2405:8100::/32;
allow 2a06:98c0::/29;
allow 2c0f:f248::/32;
deny all;

include /etc/nginx/conf.d/headers.inc;

location /api/ {
    limit_req zone=api burst=20 nodelay;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $request_id;
}

location /public/ {
    limit_req zone=api burst=20 nodelay;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $request_id;
}

# 仅代理 /admin/v1 API 节点（与仓库基线 nginx.conf 一致）；/admin/* 前端
# 页面路由必须回落 SPA（下方 location / 的 try_files），否则管理后台深链/刷新
# 会被整段代理给 Go 后端返回 404。
location /admin/v1/ {
    limit_req zone=admin burst=10 nodelay;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $request_id;
}

include /etc/nginx/conf.d/proxy.inc;

include /etc/nginx/conf.d/amap.inc;

# ── 分享链接爬虫预览（OG meta，登记例外）────────────────────────
# 默认走 SPA fallback（下方同款 try_files，用户打开 /share/[code] 页面）；
# 爬虫 UA 命中 $share_crawler（map 定义于 nginx.prod-tls.conf）时 if 内
# proxy_pass 覆盖默认分支，转给后端 GET /share/{code} 极简 HTML（og 标签）。
location /share/ {
    limit_req zone=api burst=20 nodelay;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $request_id;
    if ($share_crawler) {
        proxy_pass http://backend;
    }
    try_files $uri $uri/ /index.html;
}

location /_app/ {
    include /etc/nginx/conf.d/headers.inc;
    add_header Cache-Control "public, immutable, max-age=31536000";
    try_files $uri =404;
}

location / {
    include /etc/nginx/conf.d/headers.inc;
    add_header Cache-Control "no-cache";
    try_files $uri $uri/ /index.html;
}
