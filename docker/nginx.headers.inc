# ── 共享安全响应头（dev/prod 单一事实来源）──────────────────────────
# 由 docker/nginx.conf 与 docker/nginx.dev.conf 在 server 块内 include。
# 注意 nginx 的 add_header 不跨层级继承：location 内一旦定义 add_header，
# 本层及以下的安全头全部丢弃——因此带自定义 add_header 的 location
# （如 /_app/ 的缓存头）必须再次 include 本文件。
# 各环境专有头（prod 的 CSP、HSTS 决策等）留在各自 conf。
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
