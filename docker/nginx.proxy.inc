# ── 共享代理 location（prod 事实来源）────────────────────────────────
# 由 docker/nginx.conf 在 server 块内 include。/api/（prod 含限流）与
# location /（静态 vs Vite 代理）属本质差异，留在各自 conf。
# 注意：本地 dev 不挂载本文件，而是挂载 docker/nginx.dev.proxy.inc——
# 内容同构，差异仅两处：①OSM 上游（tile.openstreetmap.org 在国内直连不可达，
# dev 换用国内可达镜像）；②dev 不挂载 csp.inc，故 dev 的 /s3/ 只 include
# headers.inc（生产这两份片段都 include）。**瓦片与 /s3/ 两个 location 的改动
# 必须两文件同步**（缓存 key、Cache-Control、超时、安全前提注释都按同一口径）。

# ── WebSocket ─────────────────────────────────────────────
# Host 用 $http_host（含端口）而非 $host：后端 upgrader 按 Origin 裸 host
# 比对做 CSWSH 防护，$host 丢掉非标端口后（如 localhost:8888 → localhost）
# 与 Origin 对不上，握手 403（线上实测 dev 网关）。与 /s3/ 同一陷阱。
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

# ── SeaweedFS S3（presigned GET 媒体对象：头像/瞬间图/聊天文件）──────────
# 用 $http_host（含端口）而非 $host（不含端口）：presigned URL 的 SigV4 签名
# 把 host 头（如 localhost:8888 / chaoset.com）纳入签名，nginx 的 $host 会丢掉
# 端口导致验签失败。与 /ws/ 同一陷阱，不得改成 $host。
#
# ── 为什么 cache key 不含 presigned query ──────────────────────────
# 每次（重新）签发的读 URL 的 X-Amz-* query（X-Amz-Date / X-Amz-Expires /
# X-Amz-Credential / X-Amz-SignedHeaders / X-Amz-Signature）组合都不同。若 query
# 进 cache key，同一对象的每次请求都是新 key，缓存永不命中，每个头像/图片都要
# 回源 SeaweedFS（回源往返 + 磁盘读取），首屏图片一多就被放大。故 key 只用
# 「方法 + host + 归一化 URI（对象路径）」，忽略 query。
#
# ── 该做法的安全前提（任一不成立就必须收紧或关闭本 location 的缓存）──
# 1) 对象路径不可枚举：media 的 storage key 形如
#    uploads/<userID>/<uploadID>.<ext>，uploadID 由 platform/idgen 以 96bit
#    crypto/rand 生成（见 backend-go/internal/platform/idgen），路径整体不可
#    猜测；
# 2) 对象发布后内容不可变：media.Complete 的元数据/EXIF 清洗（scrub）在资产
#    登记前完成（GET→清洗→PUT 覆盖同一 key 只发生在发布前）；发布后重新上传
#    会生成新 uploadID（新 key），不存在「同一路径内容事后被换掉」的场景；
# 3) 命中缓存时不再校验签名：nginx 本身不验签（SigV4 验签在 SeaweedFS），
#    cache hit 直接返回内容，等价于把「对象路径」当作可读凭证。对头像/瞬间图
#    这类本就以 URL 在页面与客户端传播的媒体可接受；若将来出现严格私有对象
#    （一次性授权、禁止转发的文件），必须为它单开 location 并 proxy_cache off；
# 4) 只缓存整对象 200（403 签名错误 / 404 / 206 分片都不落盘，见下方指令），
#    避免脏条目污染与越权缓存；
# 5) TTL 1h：对象删除/保留期回收（retention sweep）后最多再被缓存服务 1h；
#    签名有效期（头像 15min / 瞬间图 7d）不再是内容可访问性的上界，这是本策略
#    接受的成本，也是 TTL 取 1h 而非更长/immutable 的原因。
#
# ── 注意：收益主要在源站（nginx）侧；浏览器/CF 边缘命中仍受 URL 轮换限制 ──
# 浏览器与 Cloudflare 的 cache key 包含 query：后端 presign 缓存会让同一签名
# URL 在窗口内（头像约 15min、瞬间图约 1h）复用，这段时间内下游可命中；URL
# 重新签发后即 miss。要让 CF 边缘稳定缓存 /s3/*，还需在 CF 控制台配 Cache Rule
# （Ignore query string / 自定义 cache key），属服务器侧不入库配置。本 location
# 的确定性收益：同一对象每小时最多回源 SeaweedFS 一次，其余由 nginx 本地磁盘
# 缓存直出；同时下发的 Cache-Control 让 CF/浏览器在 URL 有效窗口内具备可缓存性。
location /s3/ {
    proxy_pass http://seaweedfs_s3/;
    # 复用 upstream keepalive 池（各 nginx 主 conf 的 upstream seaweedfs_s3 已配
    # keepalive 32）：必须 HTTP/1.1 且清空 Connection 头，否则每条媒体请求都新建
    # TCP 连接（keepalive 不生效）。
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # ── 磁盘缓存（http 层 proxy_cache_path s3_media，见各 nginx 主 conf）──
    proxy_cache s3_media;
    # 只按对象路径缓存，忽略 presigned query（安全前提见上方注释）。
    proxy_cache_key "$request_method$host$uri";
    # 只缓存 GET 响应；HEAD 不缓存（S3 的 HEAD 签名语义与 GET 不同，且无 body）。
    proxy_cache_methods GET;
    # 关闭 HEAD→GET 转换（默认 on）：否则一条 HEAD 请求会让 nginx 回源下载整个
    # 对象（大媒体白下载一次）。HEAD 直接透传给 SeaweedFS。
    proxy_cache_convert_head off;
    # TTL 由代理侧决定：忽略上游 SeaweedFS 的 Cache-Control/Expires/Set-Cookie
    # （其默认 Cache-Control: max-age=14400 会让缓存期不受控），统一 1h；
    # 404 只短存，避免错误/半成品对象长期占位。
    proxy_ignore_headers Cache-Control Expires Set-Cookie;
    proxy_cache_valid 200 1h;
    proxy_cache_valid 404 60s;
    # 同一对象并发未命中只回源一次（首屏多图同对象/多标签页场景防击穿）。
    proxy_cache_lock on;
    proxy_cache_lock_timeout 10s;
    # 上游抖动/更新中可继续用旧副本，避免首屏图片 5xx。
    proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
    # Range 请求显式绕过缓存：nginx 未开 slice/force_ranges 时无法用整对象缓存
    # 服务 206，且 206 若以同一 key 落盘会污染后续整对象读取。缓存只放整对象。
    proxy_cache_bypass $http_range;
    proxy_no_cache $http_range;
    # 覆盖为可缓存的 Cache-Control（先隐藏上游的）：供 CF/浏览器按需缓存；
    # 与代理侧 TTL 对齐 1h。不是 immutable——presigned URL 会被重新签发，
    # 且删除/回收后的可访问窗口应与 TTL 一致。
    proxy_hide_header Cache-Control;
    proxy_hide_header Expires;
    add_header Cache-Control "public, max-age=3600";
    add_header X-S3-Cache $upstream_cache_status;
    # 本 location 有 add_header → server 层安全头不再继承（nginx 不跨层级继承）。
    # 与 /_app/ 同一处理：重新 include 两份片段，保住 nosniff/CORP 等（CSP 对
    # 媒体响应本身无效，但直接打开用户上传的 SVG/HTML 时仍在文档上下文生效）。
    include /etc/nginx/conf.d/headers.inc;
    include /etc/nginx/conf.d/csp.inc;

    # 超时收紧：对象存储同机内网，连不上/挂起时快速失败，不拖死首屏（头像并发多）；
    # read 放 30s 以覆盖大图/聊天文件。
    proxy_connect_timeout 3s;
    proxy_send_timeout 15s;
    proxy_read_timeout 30s;
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
