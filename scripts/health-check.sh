#!/bin/sh
# ── 生产环境健康巡检（服务器侧 crontab 调用）─────────────────────────
# 用法（crontab -e）：
#   */10 * * * * /home/workuser/hotspot/scripts/health-check.sh >> /home/workuser/hotspot/backups/health-check.log 2>&1
#
# 检查项：
#   1. 后端 health 端点可达且返回版本号（容器网络内直连，不依赖域名/CF）
#   2. 核心容器（postgres/redis/seaweedfs/backend）在运行
#   3. 磁盘水位：< 80% 正常，80-89% 告警，>= 90% 失败
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
DISK_WARN="${DISK_WARN:-80}"
DISK_FAIL="${DISK_FAIL:-90}"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"

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
  mark_fail "health FAIL: backend health 端点不可达或无版本字段（检查 docker logs $BACKEND_CONTAINER）"
fi

# 2. 核心容器运行状态
for c in "$DB_CONTAINER" "$REDIS_CONTAINER" "$SEAWEEDFS_CONTAINER" "$BACKEND_CONTAINER"; do
  if [ "$(docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null)" != "true" ]; then
    mark_fail "container FAIL: $c 未在运行"
  fi
done

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
