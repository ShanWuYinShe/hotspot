#!/bin/sh
# ── Postgres 定时备份（服务器侧 crontab 调用）────────────────────────
# 用法（crontab -e）：
#   30 3 * * * /home/workuser/hotspot/scripts/backup-postgres.sh >> /home/workuser/hotspot/backups/backup.log 2>&1
#
# 从 compose 网络内对 postgres 容器执行 pg_dump（-Z1 gzip），保留最近
# BACKUP_KEEP 份（默认 14）。备份目录在仓库根 backups/（.gitignore 已排除）。
#
# 依赖：宿主机可执行 docker（deploy 用户在 docker 组）；还原示例：
#   gunzip -c backups/hotspot_YYYYmmdd_HHMMSS.sql.gz | \
#     docker exec -i hotspot-postgres-1 psql -U postgres -d hotspot

set -eu

BACKUP_DIR="${BACKUP_DIR:-$(cd "$(dirname "$0")/.." && pwd)/backups}"
BACKUP_KEEP="${BACKUP_KEEP:-14}"
CONTAINER="${CONTAINER:-hotspot-postgres-1}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-hotspot}"

mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
TARGET="$BACKUP_DIR/hotspot_${STAMP}.sql.gz"

docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Z1 > "$TARGET"

# 只留最近 N 份，删更旧的（原子：先列后删，失败不中断主流程）
ls -1t "$BACKUP_DIR"/hotspot_*.sql.gz 2>/dev/null | tail -n +"$((BACKUP_KEEP + 1))" | while read -r old; do
	rm -f "$old"
done

echo "$(date '+%F %T') backup ok: $TARGET ($(du -h "$TARGET" | cut -f1))"
