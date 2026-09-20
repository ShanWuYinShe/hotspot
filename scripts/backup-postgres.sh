#!/bin/sh
# ── Postgres 定时备份（服务器侧 crontab 调用）────────────────────────
# 用法（crontab -e）：
#   30 3 * * * /home/workuser/hotspot/scripts/backup-postgres.sh >> /home/workuser/hotspot/backups/backup.log 2>&1
#
# 从 compose 网络内对 postgres 容器执行 pg_dump -Fc（custom format，自带
# 压缩、支持 pg_restore 选择性恢复）， dump 后立即用 pg_restore -l 读归档
# 目录校验完整性（坏备份不留着轮转掉好备份）。保留最近 BACKUP_KEEP 份
# （默认 14），同时清理历史遗留的 .sql.gz plain 格式。备份目录在仓库根
# backups/（.gitignore 已排除）。
#
# 依赖：宿主机可执行 docker（deploy 用户在 docker 组）；还原示例（runbook
# docs/backup-restore.md 第 3 节为权威）：
#   docker exec -i hotspot-postgres-1 pg_restore -U postgres -d hotspot \
#     --no-owner --clean --if-exists < backups/hotspot_YYYYmmdd_HHMMSS.dump
#
# 告警：ALERT_WEBHOOK 配置为接收 POST JSON {"text":...} 的地址（同
# health-check.sh），备份或校验失败时推送；留空 = 仅记录与退出码。

set -eu

BACKUP_DIR="${BACKUP_DIR:-$(cd "$(dirname "$0")/.." && pwd)/backups}"
BACKUP_KEEP="${BACKUP_KEEP:-14}"
CONTAINER="${CONTAINER:-hotspot-postgres-1}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-hotspot}"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"

alert() {
	[ -n "$ALERT_WEBHOOK" ] || return 0
	ESCAPED="$(printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')"
	curl -sf -m 10 -H 'Content-Type: application/json' \
		-d "{\"text\": \"$ESCAPED\"}" "$ALERT_WEBHOOK" >/dev/null 2>&1 \
		|| echo "$(date '+%F %T') alert webhook 推送失败"
}

mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
TARGET="$BACKUP_DIR/hotspot_${STAMP}.dump"

fail() {
	echo "$(date '+%F %T') backup FAIL: $1"
	alert "HotSpot Postgres 备份失败：$1"
	exit 1
}

docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc --no-owner > "$TARGET" \
	|| fail "pg_dump 执行失败（检查容器与磁盘）"

# 空文件 / 截断文件拦截：pg_restore -l 只读归档目录（TOC），损坏即非零退出
if [ ! -s "$TARGET" ]; then
	rm -f "$TARGET"
	fail "dump 为空文件，已删除"
fi
docker exec -i "$CONTAINER" pg_restore -l >/dev/null < "$TARGET" \
	|| fail "pg_restore -l 完整性校验未通过（dump 损坏），已保留现场供排查: $TARGET"

# 异地副本（可选）：服务器磁盘故障时本地轮转救不了备份——配置 OFFSITE_CMD
# 即在轮转前上传，{} 占位符替换为本次备份路径。示例（写入 crontab 环境）：
#   OFFSITE_CMD='rclone copy {} oss:hotspot-backups'
#   OFFSITE_CMD='scp {} backup@nas:/volume1/hotspot-backups'
# 上传失败按备份失败处理（告警 + 非零退出），本地文件保留待下次轮转。
if [ -n "${OFFSITE_CMD:-}" ]; then
	if printf '%s' "$OFFSITE_CMD" | sed "s|{}|\"$TARGET\"|g" | sh >/dev/null 2>&1; then
		echo "$(date '+%F %T') backup offsite ok: $OFFSITE_CMD"
	else
		fail "异地上传失败（OFFSITE_CMD），本地保留: $TARGET"
	fi
fi

# 只留最近 N 份（新 .dump 与历史遗留 .sql.gz 一并轮转）
ls -1t "$BACKUP_DIR"/hotspot_*.dump "$BACKUP_DIR"/hotspot_*.sql.gz 2>/dev/null | tail -n +"$((BACKUP_KEEP + 1))" | while read -r old; do
	rm -f "$old"
done

echo "$(date '+%F %T') backup ok: $TARGET ($(du -h "$TARGET" | cut -f1))"
