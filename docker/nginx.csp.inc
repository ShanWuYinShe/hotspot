# ── CSP（仅生产启用；dev 不 include）────────────────────────────────
# 为什么独立成文件：
#  - dev 网关（docker/nginx.dev.conf）**不加 CSP**——Vite HMR 需要 inline script/eval
#    与 blob worker（开发工具链），且 dev 只服务本机调试；此前 CSP 写在 headers.inc 里
#    被 dev 一并继承，与 dev 配置注释「不加 CSP」矛盾，并会在控制台报
#    「blocked worker-src blob:…」等无关噪声（2026-10-31 修正）。
#  - 生产必须带 CSP：生产 conf（nginx.conf / prod-server.inc）在每个 include 了
#    headers.inc 的位置**紧跟**本文件——nginx 的 add_header 不跨层级继承，HTML
#    入口（location / 与 /_app/）带自定义缓存头会覆盖 server 层安全头，因此这些
#    location 内要同时重新 include 两份片段。
#
#  - script-src 'unsafe-inline'：SvelteKit 模块引导需内联 bootstrap script
#    （<script type="module">import { start } ...），被 'self' 拦截会导致
#    生产白屏；无第三方注入面，风险可控
#  - script-src 放行 static.cloudflareinsights.com、connect-src 放行
#    cloudflareinsights.com：橙云站点由 CF 边缘自动注入 Web Analytics
#    beacon（RUM 统计），不放行会被 CSP 拦截并在控制台报 CSP 违规；除 CF
#    自动注入外，站点自身不加载任何第三方脚本
#  - style-src 'unsafe-inline'：Tailwind / Svelte 注入内联样式
#  - img-src：允许 data: URI 与任意 https CDN（用户头像/地图瓦片）
#  - connect-src 'self'：API/WS 必须同源；restapi.amap.com 为高德 Web 服务
#    API（地理编码/POI 搜索直连，见 lib/geo-service.ts）。当前未配置 key 时
#    不会向其发请求，放行是为了配置 key 后不被 CSP 拦截退回演示数据
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' https://static.cloudflareinsights.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https://restapi.amap.com https://cloudflareinsights.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self'" always;
