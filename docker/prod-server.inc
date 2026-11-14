# ── HotSpot 生产 443 server{} 公共片段（仅 443 使用）────────────
# 位置类配置与仓库 docker/nginx.conf 保持一致；其中「首屏静态资源缓存策略」
# 四段（/fonts/、/icons/、= /favicon.png、= /manifest.webmanifest）与
# docker/nginx.conf 的同段**逐字一致**，任一侧改动都要同步另一侧。/s3/、瓦片等
# 共享代理 location 抽在 docker/nginx.proxy.inc（base 与 443 共用同一挂载文件）。
# 依赖同目录挂载的 headers.inc / csp.inc / proxy.inc / amap.inc。
# 80 端口端点定义在 nginx.prod-tls.conf。
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
include /etc/nginx/conf.d/csp.inc;

# ── 限流超限统一响应 ──────────────────────────────────────────
# limit_req 默认返回 503 HTML；改为 429 + 与后端同形的 {"code":10500,...}
# 响应体 + Retry-After，前端 core.ts 会按 Retry-After（≤10s）自动重试一次。
# proxy_intercept_errors 默认为 off，后端自己返回的 429（WS 握手）原样透传。
error_page 429 = @rate_limited;
location @rate_limited {
    default_type application/json;
    add_header Retry-After 1 always;
    add_header Cache-Control "no-store" always;
    return 429 '{"code":10500,"message":"请求过于频繁"}';
}

# 存活探针（全站约定 GET /healthz）：无鉴权、无限流，直通后端。
location = /healthz {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
}

location /api/ {
    limit_req zone=api burst=20 nodelay;
    # JSON API 体量上限：后端 MaxRequestBodyBytes=1MB，nginx 只兜住极端值
    # （server 级 100m 留给 /s3/ 直传）。
    client_max_body_size 2m;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
}

# ── 头像代理（/public/v1/media/avatars/{file_id} → 302 签名 URL）──
# 单独限流桶：单页可能一口气加载几十张头像，共享 api 桶（burst 20）会误伤
# （2026-11-08 受控实测 60 并发 → 22 通过 / 38×503）；这是廉价 302，给更宽的桶。
location /public/v1/media/avatars/ {
    limit_req zone=avatar burst=60 nodelay;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
}

location /public/ {
    limit_req zone=api burst=20 nodelay;
    client_max_body_size 2m;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
}

# 仅代理 /admin/api/v1 API 节点（与仓库基线 nginx.conf 一致）；/admin/* 前端
# 页面路由必须回落 SPA（下方 location / 的 try_files），否则管理后台深链/刷新
# 会被整段代理给 Go 后端返回 404。
location /admin/api/v1/ {
    limit_req zone=admin burst=10 nodelay;
    client_max_body_size 2m;
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
}

include /etc/nginx/conf.d/proxy.inc;

include /etc/nginx/conf.d/amap.inc;

# ── 分享链接爬虫预览（OG meta，登记例外）────────────────────────
# 默认走 SPA fallback（下方同款 try_files，用户打开 /share/[code] 页面）；
# 爬虫 UA 且未带 ?spa=1（$share_og=1，map 定义于 nginx.prod-tls.conf）时
# if 内 proxy_pass 覆盖默认分支，转给后端 GET /share/{code} 极简 HTML（og 标签）。
# 带 ?spa=1 一律走 SPA（含爬虫 UA）：微信/QQ 内嵌浏览器与预览抓取器共用 UA
# 子串，OG 页 CTA 依赖该参数跳出爬虫分支，否则 SPA 分享页永远打不开。
location /share/ {
    limit_req zone=api burst=20 nodelay;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Request-ID $hotspot_request_id;
    if ($share_og) {
        proxy_pass http://backend;
    }
    try_files $uri $uri/ /index.html;
}

location /_app/ {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    add_header Cache-Control "public, immutable, max-age=31536000";
    try_files $uri =404;
}

# ── 首屏静态资源缓存策略（本段与 docker/nginx.conf 的同段内容必须一致：
#    只允许缩进不同，指令/注释/顺序一律相同；任一侧改动都要同步另一侧）────
# 每个 location 都重新 include headers/csp：nginx 的 add_header 不跨层级
# 继承，加了缓存头就会丢掉 server 层安全头（与 /_app/ 同一处理，见
# docker/nginx.headers.inc、docker/nginx.csp.inc 头注释）。
#
# /fonts/：字体 public, max-age=1y + immutable。当前字体文件名不带 content
# hash（frontend/static/fonts/inter-latin-wght.woff2 原样发布），用 1y 的前提
# 是「不原地替换同名文件」；将来要同文件名换内容，必须先改名（或迁到带 hash
# 的路径），否则已缓存客户端 1 年内看不到新字体。取舍：首屏 preload 字体命中
# 本地缓存零往返，代价是改名纪律。
location /fonts/ {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    add_header Cache-Control "public, max-age=31536000, immutable";
    try_files $uri =404;
}

# /icons/ 与 /favicon.png：PWA 图标/站点图标会随发版原地替换（同名），
# 故 30d 而非 immutable——换图后最多 30 天收敛，不会出现 1 年不更新。
location /icons/ {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    add_header Cache-Control "public, max-age=2592000";
    try_files $uri =404;
}

location = /favicon.png {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    add_header Cache-Control "public, max-age=2592000";
    try_files $uri =404;
}

# = /manifest.webmanifest：PWA 清单。nginx 内置 mime.types 未收录
# .webmanifest（nginx 1.30.4 实测），不显式 default_type 会以
# application/octet-stream 下发，影响 PWA 安装判定。缓存 1h：与 index.html
# 同属「改版后应尽快生效」的入口元数据，同时吸收首屏重复请求。
location = /manifest.webmanifest {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    default_type application/manifest+json;
    add_header Cache-Control "public, max-age=3600";
    try_files $uri =404;
}

# dev 网关（docker/nginx.dev.conf）前端是 Vite dev server 直出、没有构建
# 产物 root，故不做上述长缓存（有意差异，说明见该文件）。

location / {
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;
    add_header Cache-Control "no-cache";
    try_files $uri $uri/ /index.html;
}
