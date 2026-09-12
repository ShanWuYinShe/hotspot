# ── 高德 Web 服务同源代理（默认占位）──────────────────────
# 本文件经 compose 挂载为 /etc/nginx/conf.d/amap.inc（dev/prod 共用挂载点）。
#
# 前端地理编码统一走同源路径 /amap-service/（如 /amap-service/v3/geocode/regeo），
# 由本文件决定转发行为。密钥（key/jscode）只在服务器侧注入，前端零密钥：
#
#   生产：在服务器维护不入库的 docker/amap.prod.inc（与 cert/ 同模式，权限 600），
#         覆盖为真实转发（key/jscode 由 nginx 追加进 query），内容形如：
#
#             location /amap-service/ {
#                 limit_req zone=api burst=20 nodelay;
#                 proxy_ssl_server_name on;
#                 proxy_set_header Host restapi.amap.com;
#                 set $amap_args "${args}&key=<KEY>&jscode=<JSCODE>";
#                 proxy_pass https://restapi.amap.com$uri?$amap_args;
#             }
#
#   默认（本文件）：未配置密钥时返回 503 JSON——前端按「服务不可用」降级，
#   地图源切到 OSM（/osm-service/ Nominatim 无需密钥）。
location /amap-service/ {
    default_type application/json;
    return 503 '{"code":10600,"message":"高德地图服务未配置"}';
}
