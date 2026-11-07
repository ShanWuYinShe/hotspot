#!/usr/bin/env bash
# ── 异地备份拉取（在运维本机执行）────────────────────────────────
# 服务器 cron 只把备份写到本机磁盘（服务器故障即丢）。本脚本把服务器最新
# Postgres dump 与 SeaweedFS 归档拉到运维机，形成离线副本。
#
# 用法：
#   ./scripts/backup-pull.sh                 # 默认 ~/work/hotspot-backups，保留 7 份
#   DEST=/Volumes/Backup/hotspot ./scripts/backup-pull.sh
#   SERVER=workwork-server KEEP=14 ./scripts/backup-pull.sh
#
# 建议在本机定时执行（macOS launchd / cron），拉取间隔决定异地 RPO 上限。
set -euo pipefail

SERVER="${SERVER:-workwork-server}"
REMOTE_BACKUP_DIR="${REMOTE_BACKUP_DIR:-~/hotspot/backups}"
DEST="${DEST:-$HOME/work/hotspot-backups}"
KEEP="${KEEP:-7}"

mkdir -p "$DEST"

echo "==> 同步服务器备份到 $DEST …"
rsync -a --exclude '*.log' "${SERVER}:${REMOTE_BACKUP_DIR}/" "$DEST/"

# 本地保留策略：每类只留最近 KEEP 份（备份文件小，同步全量后再轮转）
for pattern in 'hotspot_*.dump' 'seaweedfs_*.tar.gz'; do
  # shellcheck disable=SC2086  # 通配符由本机展开
  ls -1t "$DEST"/$pattern 2>/dev/null | tail -n +"$((KEEP + 1))" | while read -r old; do
    rm -f "$old"
  done
done

latest_dump="$(ls -1t "$DEST"/hotspot_*.dump 2>/dev/null | head -1 || true)"
if [ -z "$latest_dump" ]; then
  echo "!! 未拉到任何 hotspot_*.dump（检查 SERVER/REMOTE_BACKUP_DIR 与 SSH 权限）" >&2
  exit 1
fi
echo "✅ 异地副本就绪：${latest_dump}（$(($(date +%s) - $(stat -f %m "$latest_dump" 2>/dev/null || stat -c %Y "$latest_dump")))s 前）"
ls -lh "$DEST" | tail -"$((KEEP > 5 ? 5 : KEEP))" || true
