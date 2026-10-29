# ── HotSpot 生产 443 server{} 公共片段（仅 443 使用）────────────
# 位置类配置与仓库 docker/nginx.conf 保持一致；依赖同目录挂载的
# headers.inc / proxy.inc / amap.inc。80 端口端点定义在 nginx.prod-tls.conf。
#
# 入口访问控制：本片段只挂在 443 server 上，来源鉴权由 TLS 层的 Cloudflare
# Authenticated Origin Pulls（mTLS）完成——nginx 握手阶段即校验 CF 回源客户端
# 证书（ssl_verify_client on，见 nginx.prod-tls.conf），直连无证书一律 400，
# 请求不会进入本片段。因此不再维护 CF 回源网段 allow/deny 白名单：旧白名单与
# real_ip 还原互斥（CF-Connecting-IP 改写后的 $remote_addr 是真实访客 IP，
# 永远匹配不上 CF 网段），是 2026-10-30 全站 403 事故的根因。

root /usr/share/nginx/html;
client_max_body_size 100m;
server_tokens off;

include /etc/nginx/conf.d/headers.inc;

# 存活探针（全站约定 GET /healthz）：无鉴权、无限流，直通后端。
location = /healthz {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $request_id;
}

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

# 仅代理 /admin/api/v1 API 节点（与仓库基线 nginx.conf 一致）；/admin/* 前端
# 页面路由必须回落 SPA（下方 location / 的 try_files），否则管理后台深链/刷新
# 会被整段代理给 Go 后端返回 404。
location /admin/api/v1/ {
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
