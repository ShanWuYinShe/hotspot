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
# 仅记录与退出码，不做自动恢复（自动重启由 compose restart: always 兜底）。

set -u

REMOTE_DIR="${REMOTE_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
BACKEND_CONTAINER="${BACKEND_CONTAINER:-hotspot-backend-1}"
DB_CONTAINER="${DB_CONTAINER:-hotspot-postgres-1}"
REDIS_CONTAINER="${REDIS_CONTAINER:-hotspot-redis-1}"
SEAWEEDFS_CONTAINER="${SEAWEEDFS_CONTAINER:-hotspot-seaweedfs-1}"
DISK_WARN="${DISK_WARN:-80}"
DISK_FAIL="${DISK_FAIL:-90}"

fail=0

# 1. 后端 health（版本字段存在即认为服务可用）
HEALTH="$(docker exec "$BACKEND_CONTAINER" curl -sf -X POST -H 'Content-Type: application/json' \
  -d '{}' http://localhost:8081/public/v1/system/health 2>/dev/null || true)"
if echo "$HEALTH" | grep -q '"version"'; then
  echo "$(date '+%F %T') health ok: $(echo "$HEALTH" | head -c 200)"
else
  echo "$(date '+%F %T') health FAIL: backend health 端点不可达或无版本字段（检查 docker logs $BACKEND_CONTAINER）"
  fail=1
fi

# 2. 核心容器运行状态
for c in "$DB_CONTAINER" "$REDIS_CONTAINER" "$SEAWEEDFS_CONTAINER" "$BACKEND_CONTAINER"; do
  if [ "$(docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null)" != "true" ]; then
    echo "$(date '+%F %T') container FAIL: $c 未在运行"
    fail=1
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
    echo "$(date '+%F %T') disk FAIL: $p 使用率 ${usage}%（>= ${DISK_FAIL}%，请清理 docker image / 旧备份 / 瓦片缓存）"
    fail=1
  elif [ "$usage" -ge "$DISK_WARN" ]; then
    echo "$(date '+%F %T') disk WARN: $p 使用率 ${usage}%（${DISK_WARN}-${DISK_FAIL}% 区间）"
  else
    echo "$(date '+%F %T') disk ok: $p 使用率 ${usage}%"
  fi
done

exit $fail
