# ── 共享代理 location（prod 事实来源）────────────────────────────────
# 由 docker/nginx.conf 在 server 块内 include。/api/（prod 含限流）与
# location /（静态 vs Vite 代理）属本质差异，留在各自 conf。
# 注意：本地 dev 不挂载本文件，而是挂载 docker/nginx.dev.proxy.inc——
# 内容同构，差异仅在 OSM 上游（tile.openstreetmap.org 在国内直连不可达，
# dev 换用 Carto 镜像）；两文件的瓦片 location 改动必须保持同步。

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

# ── 高德瓦片同源代理（地图底图）──────────────────────────
# 前端请求 /amap-tiles/{z}/{x}/{y}.png → 高德 appmaptile 栅格瓦片（GCJ-02）。
# 栅格瓦片不需要 key/jscode；同源转发避免客户端直连外网（跨域/不可达环境）。
# 瓦片不可变：代理侧落盘缓存 30 天，重复浏览零回源（此前无缓存，每次缩放
# 拖动都回源拉取，是「高德源也慢」的主因）。
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

# ── OpenStreetMap 瓦片同源代理（地图底图备选源）──────────
# OSM 公共瓦片服务要求低频访问 + 大力缓存（osm.org/tile-use-policy）：
# 代理侧 proxy_cache 落盘 30 天，同一瓦片只回源一次；Cache-Control 由
# 上游下发（X-Cache-Lite 命中态也按上游 TTL 处理）。显式设置 UA/Referer
# 便于 OSM 侧识别来源；缓存命中时不回源，天然满足使用政策。
# 连接/读取超时收紧到 5s/15s：上游不可达时快速 502（默认 60s 挂起会拖死
# 整个视口的瓦片并发加载，前端表现为整屏黑块卡死）。
# 地图源切换（高德 GCJ-02 / OSM WGS-84）见前端 mapSource store。
location /osm-tiles/ {
    rewrite ^/osm-tiles/(\d+)/(\d+)/(\d+)\.png$ /$1/$2/$3.png break;
    proxy_pass https://tile.openstreetmap.org;
    proxy_ssl_server_name on;
    proxy_set_header Host tile.openstreetmap.org;
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
# 搜索（/search）与逆地理（/reverse）。公共 Nominatim 政策：≤1 req/s、
# 必须带可识别 UA——限流 1r/s 在代理侧兜底，前端另有 debounce。
# 语言随浏览器 Accept-Language 由上游解析。
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
