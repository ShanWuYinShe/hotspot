# ── 共享代理 location（dev/prod 单一事实来源）────────────────────────
# 由 docker/nginx.conf 与 docker/nginx.dev.conf 在 server 块内 include。
# /api/（prod 含限流）与 location /（静态 vs Vite 代理）属本质差异，留在各自 conf。

# ── WebSocket ─────────────────────────────────────────────
location /ws/ {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400s;
}

# ── SeaweedFS S3 ─────────────────────────────────────────
# 用 $http_host（含端口）而非 $host（不含端口）：presigned URL 的 SigV4 签名
# 把 host 头（如 localhost:8888）纳入签名，nginx 的 $host 会丢掉端口导致验签失败。
location /s3/ {
    proxy_pass http://seaweedfs_s3/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
