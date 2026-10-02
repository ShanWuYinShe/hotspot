#!/bin/sh
# ── 生产环境健康巡检（服务器侧 crontab 调用）─────────────────────────
# 用法（crontab -e）：
#   */10 * * * * /home/workuser/hotspot/scripts/health-check.sh >> /home/workuser/hotspot/backups/health-check.log 2>&1
#
# 检查项：
#   1. 后端 health 端点可达且返回版本号（容器网络内直连，不依赖域名/CF）
#   2. 核心容器（postgres/redis/seaweedfs/backend/nginx/frontend）在运行
#   3. 磁盘水位：< 80% 正常，80-89% 告警，>= 90% 失败
#   4. 备份新鲜度：backups/ 下最新 .sql.gz/.dump 距今 < BACKUP_MAX_AGE_HOURS（默认 25h）
#   5. TLS 证书剩余有效期 >= CERT_MIN_DAYS（默认 14 天；读 nginx 挂载的证书文件）
#
# 退出码：0 = 全部正常；1 = 存在失败项（异常时查看本日志）。
# 告警送达：ALERT_WEBHOOK 配置为接收 POST JSON 的地址（飞书/钉钉/企微
# 自定义机器人通用 {"text": "..."} 格式；钉钉加签等专有格式请按需适配），
# 存在失败项时推送摘要；留空 = 不告警，仅记录与退出码。
# 仅记录与告警，不做自动恢复（自动重启由 compose restart: always 兜底）。

set -u

REMOTE_DIR="${REMOTE_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
BACKEND_CONTAINER="${BACKEND_CONTAINER:-hotspot-backend-1}"
DB_CONTAINER="${DB_CONTAINER:-hotspot-postgres-1}"
REDIS_CONTAINER="${REDIS_CONTAINER:-hotspot-redis-1}"
SEAWEEDFS_CONTAINER="${SEAWEEDFS_CONTAINER:-hotspot-seaweedfs-1}"
NGINX_CONTAINER="${NGINX_CONTAINER:-hotspot-nginx-1}"
FRONTEND_CONTAINER="${FRONTEND_CONTAINER:-hotspot-frontend-1}"
DISK_WARN="${DISK_WARN:-80}"
DISK_FAIL="${DISK_FAIL:-90}"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"
BACKUP_DIR="${BACKUP_DIR:-$REMOTE_DIR/backups}"
BACKUP_MAX_AGE_HOURS="${BACKUP_MAX_AGE_HOURS:-25}"
# 证书路径：compose.prod 把服务器 $REMOTE_DIR/cert/ 挂到 nginx /etc/nginx/certs，
# 证书文件名与 docker/nginx.prod-tls.conf 的 ssl_certificate 一致；空 = 跳过检查
CERT_FILE="${CERT_FILE:-$REMOTE_DIR/cert/chaoset.pem}"
CERT_MIN_DAYS="${CERT_MIN_DAYS:-14}"

fail=0
fail_detail=""

# 失败项统一记录：写日志 + 累积明细（供告警摘要）
mark_fail() {
  echo "$(date '+%F %T') $1"
  fail=1
  fail_detail="$fail_detail\n$1"
}

# 1. 后端 health（版本字段存在即认为服务可用）
HEALTH="$(docker exec "$BACKEND_CONTAINER" curl -sf -X POST -H 'Content-Type: application/json' \
  -d '{}' http://localhost:8081/public/v1/system/health 2>/dev/null || true)"
if echo "$HEALTH" | grep -q '"version"'; then
  echo "$(date '+%F %T') health ok: $(echo "$HEALTH" | head -c 200)"
else
  mark_fail "health FAIL: backend health 端点不可达或无版本字段（检查 docker logs ${BACKEND_CONTAINER}）"
fi

# 2. 核心容器运行状态（含 nginx 网关与 frontend 静态卷容器——此前巡检不覆盖，
#    网关挂了站点全断但巡检仍报绿）
for c in "$DB_CONTAINER" "$REDIS_CONTAINER" "$SEAWEEDFS_CONTAINER" "$BACKEND_CONTAINER" "$NGINX_CONTAINER" "$FRONTEND_CONTAINER"; do
  if [ "$(docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null)" != "true" ]; then
    mark_fail "container FAIL: $c 未在运行"
  fi
done

# 2.5 备份新鲜度：坏备份静默轮转掉好备份是巡检盲区——最新备份超龄即失败。
latest_backup="$(ls -1t "$BACKUP_DIR"/*.sql.gz "$BACKUP_DIR"/*.dump 2>/dev/null | head -1 || true)"
if [ -z "$latest_backup" ]; then
  mark_fail "backup FAIL: $BACKUP_DIR 下没有任何备份文件"
elif [ "$(find "$latest_backup" -mmin +$((BACKUP_MAX_AGE_HOURS * 60)) 2>/dev/null | wc -l)" -gt 0 ]; then
  mark_fail "backup FAIL: 最新备份 $(basename "$latest_backup") 已超过 ${BACKUP_MAX_AGE_HOURS}h（检查备份 cron 是否在跑）"
else
  echo "$(date '+%F %T') backup ok: 最新备份 $(basename "$latest_backup")"
fi

# 2.6 TLS 证书剩余有效期（Cloudflare Origin CA，年均轮换；临近过期提前告警）
if [ -n "$CERT_FILE" ] && [ -f "$CERT_FILE" ]; then
  if command -v openssl >/dev/null 2>&1; then
    end_ts="$(openssl x509 -in "$CERT_FILE" -noout -enddate 2>/dev/null | cut -d= -f2)"
    if [ -n "$end_ts" ]; then
      days_left=$(( ($(date -d "$end_ts" +%s 2>/dev/null || date -j -f '%b %e %T %Y %Z' "$end_ts" +%s 2>/dev/null) - $(date +%s)) / 86400 ))
      if [ "$days_left" -lt "$CERT_MIN_DAYS" ]; then
        mark_fail "cert FAIL: $CERT_FILE 剩余 ${days_left} 天（< ${CERT_MIN_DAYS}），请轮换 Cloudflare Origin CA 证书"
      else
        echo "$(date '+%F %T') cert ok: $CERT_FILE 剩余 ${days_left} 天"
      fi
    else
      mark_fail "cert FAIL: 无法解析 $CERT_FILE 有效期（openssl x509 失败，文件可能损坏）"
    fi
  fi
elif [ -n "$CERT_FILE" ]; then
  mark_fail "cert FAIL: 证书文件不存在: $CERT_FILE"
fi

# 3. 磁盘水位（覆盖根分区；数据盘另挂载时把路径追加到 DISK_PATHS）
DISK_PATHS="${DISK_PATHS:-/}"
for p in $DISK_PATHS; do
  usage="$(df -P "$p" | awk 'NR==2 {gsub("%",""); print $5}')"
  if [ -z "$usage" ]; then
    echo "$(date '+%F %T') disk WARN: 无法读取 $p 使用率"
    continue
  fi
  if [ "$usage" -ge "$DISK_FAIL" ]; then
    mark_fail "disk FAIL: $p 使用率 ${usage}%（>= ${DISK_FAIL}%，请清理 docker image / 旧备份 / 瓦片缓存）"
  elif [ "$usage" -ge "$DISK_WARN" ]; then
    echo "$(date '+%F %T') disk WARN: $p 使用率 ${usage}%（${DISK_WARN}-${DISK_FAIL}% 区间）"
  else
    echo "$(date '+%F %T') disk ok: $p 使用率 ${usage}%"
  fi
done

# 告警送达：存在失败项且配置了 webhook 时推送摘要（推送失败不影响退出码）。
if [ "$fail" -ne 0 ] && [ -n "$ALERT_WEBHOOK" ]; then
  SUMMARY="$(hostname) HotSpot 巡检失败：$(printf '%b' "$fail_detail")"
  # JSON 字符串转义：反斜杠与双引号，换行压成空格
  ESCAPED="$(printf '%s' "$SUMMARY" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')"
  curl -sf -m 10 -H 'Content-Type: application/json' \
    -d "{\"text\": \"$ESCAPED\"}" "$ALERT_WEBHOOK" \
    >/dev/null 2>&1 || echo "$(date '+%F %T') alert webhook 推送失败"
fi

exit $fail
