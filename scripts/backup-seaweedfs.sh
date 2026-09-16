#!/bin/sh
# ── SeaweedFS 媒体卷定时备份（服务器侧 crontab 调用）──────────────────
# 用法（crontab -e，与 backup-postgres.sh 错开时间）：
#   0 4 * * * /home/workuser/hotspot/scripts/backup-seaweedfs.sh >> /home/workuser/hotspot/backups/backup.log 2>&1
#
# 将 seaweedfs_data 卷（全部上传的图片/视频）tar 打包到 backups/，保留最近
# BACKUP_KEEP 份（默认 7，媒体体积大、密度低于数据库）。备份经一次性只读
# 挂载容器执行，不写入业务卷、不依赖 seaweedfs 容器内工具链。
#
# 依赖：宿主机可执行 docker；首次运行会拉取 BACKUP_IMAGE（alpine）。
# 还原示例：
#   docker run --rm -v hotspot_seaweedfs_data:/data -v "$PWD":/backup alpine:3.22 \
#     sh -c 'cd /data && tar xzf /backup/seaweedfs_YYYYmmdd_HHMMSS.tar.gz --strip-components=1'

set -eu

BACKUP_DIR="${BACKUP_DIR:-$(cd "$(dirname "$0")/.." && pwd)/backups}"
BACKUP_KEEP="${BACKUP_KEEP:-7}"
# compose 项目名 = 目录名；服务器部署目录约定为 ~/hotspot → hotspot_seaweedfs_data
VOLUME="${VOLUME:-hotspot_seaweedfs_data}"
BACKUP_IMAGE="${BACKUP_IMAGE:-alpine:3.22}"

mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
TARGET="$BACKUP_DIR/seaweedfs_${STAMP}.tar.gz"

docker run --rm -v "${VOLUME}":/data:ro -v "$BACKUP_DIR":/backup "$BACKUP_IMAGE" \
  tar czf "/backup/seaweedfs_${STAMP}.tar.gz" -C / data

# 只留最近 N 份，删更旧的（失败不中断主流程）
ls -1t "$BACKUP_DIR"/seaweedfs_*.tar.gz 2>/dev/null | tail -n +"$((BACKUP_KEEP + 1))" | while read -r old; do
	rm -f "$old"
done

echo "$(date '+%F %T') seaweedfs backup ok: $TARGET ($(du -h "$TARGET" | cut -f1))"
