# ── 共享安全响应头（dev/prod 单一事实来源）──────────────────────────
# 由 docker/nginx.conf 与 docker/nginx.dev.conf 在 server 块内 include。
# 注意 nginx 的 add_header 不跨层级继承：location 内一旦定义 add_header，
# 本层及以下的安全头全部丢弃——因此带自定义 add_header 的 location
# （如 /_app/ 的缓存头）必须再次 include 本文件。
# CSP 也在本文件：HTML 入口（location / 与 /_app/）恰恰带自定义缓存头，
# 放 server 层会被继承规则丢弃——最需要 CSP 的页面反而拿不到。
# 各环境专有头（如 HSTS 决策）留在各自 conf。
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=(self), payment=(), usb=()" always;
# 跨域隔离（OWASP Secure Headers 基线）：同源部署下用户内容（/s3/）不应被
# 任意站点嵌入读取；COOP 阻断跨窗口 opener 攻击面。
add_header Cross-Origin-Opener-Policy "same-origin" always;
add_header Cross-Origin-Resource-Policy "same-origin" always;
#  - script-src 'unsafe-inline'：SvelteKit 模块引导需内联 bootstrap script
#    （<script type="module">import { start } ...），被 'self' 拦截会导致
#    生产白屏；无第三方注入面，风险可控
#  - style-src 'unsafe-inline'：Tailwind / Svelte 注入内联样式
#  - img-src：允许 data: URI 与任意 https CDN（用户头像/地图瓦片）
#  - connect-src 'self'：API/WS 必须同源；如需接外部 API 在此追加
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'" always;
