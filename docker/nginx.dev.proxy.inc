# ── 共享代理 location（dev 版）───────────────────────────────────────
# 与 docker/nginx.proxy.inc（prod 事实源）内容同构，唯一差异：
# OSM 官方瓦片源 tile.openstreetmap.org 在国内网络直连不可达（容器内实测
# TCP 全超时 / TLS RST / DNS 污染），本地 dev 曾换用 CARTO basemaps 镜像
# ——2026-09-26 起 CARTO 对无 API key 请求返回「API KEY REQUIRED」占位
# 瓦片（HTTP 200，ETag 前缀 wm-），该源不可用。现换用 FOSSGIS 的
# tile.openstreetmap.de（OSM 数据、免 key、国内直连实测可达；usage policy
# 要求合理用量与署名，attribution 已在前端瓦片层展示
# © OpenStreetMap contributors）。Nominatim（/osm-service/）没有等价公开
# 镜像，dev 保持官方上游——不可达时地理编码降级失败，瓦片显示不受影响。
# 两个文件的瓦片 location 改动必须保持同步。
# 由 docker-compose.dev.yml 挂载为 /etc/nginx/conf.d/proxy.inc，
# 由 docker/nginx.dev.conf 在 server 块内 include。

# ── WebSocket ─────────────────────────────────────────────
# Host 用 $http_host（含端口）而非 $host：后端 upgrader 按 Origin 裸 host
# 比对做 CSWSH 防护，$host 丢掉非标端口后（如 localhost:8888 → localhost）
# 与 Origin 对不上，握手 403（线上实测）。与 /s3/ 同一陷阱，与 prod 同构。
location /ws/ {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $http_host;
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

# ── 高德瓦片同源代理（地图底图）──────────────────────────
# 与 prod 同构：栅格瓦片不需要 key/jscode；缓存 30 天，重复浏览零回源。
location /amap-tiles/ {
    rewrite ^/amap-tiles/(\d+)/(\d+)/(\d+)\.png$ /appmaptile?lang=zh_cn&size=1&scale=1&style=7&x=$2&y=$3&z=$1 break;
    proxy_pass https://webrd04.is.autonavi.com;
    proxy_ssl_server_name on;
    proxy_set_header Host webrd04.is.autonavi.com;
    proxy_cache amap_tiles;
    proxy_cache_valid 200 30d;
    proxy_cache_valid 404 1m;
    proxy_cache_lock on;
    proxy_cache_lock_timeout 5s;
    add_header X-Tile-Cache $upstream_cache_status;
}

# ── OpenStreetMap 瓦片同源代理（dev：CARTO 镜像上游）─────
# 官方 tile.openstreetmap.org 国内直连不可达，dev 换 CARTO basemaps
# （OSM 数据 voyager 样式；OSM 底图恒等直通 WGS-84 的坐标口径不变）。
# 缓存与 UA 标识策略与 prod 一致：30 天落盘、同一瓦片只回源一次；
# 连接/读取超时收紧，上游异常快速 502 不拖死视口加载。
location /osm-tiles/ {
    rewrite ^/osm-tiles/(\d+)/(\d+)/(\d+)\.png$ /$1/$2/$3.png break;
    proxy_pass https://tile.openstreetmap.de;
    proxy_ssl_server_name on;
    proxy_set_header Host tile.openstreetmap.de;
    proxy_set_header User-Agent "HotSpot-Proxy/1.0 (chaoset.com)";
    proxy_set_header Accept-Encoding "";
    proxy_connect_timeout 5s;
    proxy_send_timeout 15s;
    proxy_read_timeout 15s;
    proxy_cache osm_tiles;
    proxy_cache_valid 200 30d;
    proxy_cache_valid 404 1m;
    proxy_cache_lock on;
    proxy_cache_lock_timeout 5s;
    add_header X-Tile-Cache $upstream_cache_status;
}

# ── OSM Nominatim 同源代理（地理编码，OSM 源配套）────────
# dev 保持官方上游（无公开镜像；国内直连不可达时前端地理编码降级失败）。
# 限流与 UA 政策与 prod 一致：≤1 req/s + 可识别 UA。
location /osm-service/ {
    limit_req zone=osm_service burst=3 nodelay;
    rewrite ^/osm-service/(.*)$ /$1 break;
    proxy_pass https://nominatim.openstreetmap.org;
    proxy_ssl_server_name on;
    proxy_set_header Host nominatim.openstreetmap.org;
    proxy_set_header User-Agent "HotSpot-Proxy/1.0 (chaoset.com)";
    proxy_connect_timeout 5s;
    proxy_send_timeout 15s;
    proxy_read_timeout 15s;
    proxy_cache off;
}
