# ── 共享安全响应头（dev/prod 单一事实来源）──────────────────────────
# 由 docker/nginx.conf 与 docker/nginx.dev.conf 在 server 块内 include。
# 注意 nginx 的 add_header 不跨层级继承：location 内一旦定义 add_header，
# 本层及以下的安全头全部丢弃——因此带自定义 add_header 的 location
# （如 /_app/ 的缓存头）必须再次 include 本文件。
# **CSP 不在本文件**：它是生产专用片段 docker/nginx.csp.inc（dev 不加 CSP，
# 原因见该文件头注释）；生产 conf 在每个 include 了本文件的位置紧接着
# include csp.inc。各环境专有头（如 HSTS 决策）留在各自 conf。
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=(self), payment=(), usb=()" always;
# 跨域隔离（OWASP Secure Headers 基线）：同源部署下用户内容（/s3/）不应被
# 任意站点嵌入读取；COOP 阻断跨窗口 opener 攻击面。
add_header Cross-Origin-Opener-Policy "same-origin" always;
add_header Cross-Origin-Resource-Policy "same-origin" always;

