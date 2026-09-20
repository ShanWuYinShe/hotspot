#!/usr/bin/env bash
# ── HotSpot 生产部署（产物分发模式）─────────────────────────────────
#
# 标准流程：git tag → CI 构建镜像并挂到 GitHub Release（.github/workflows/
# release.yml）→ 本脚本从 Release 下载镜像包、分发到服务器 docker load。
# 服务器全程不接触源码、不执行构建；本机也不再向服务器 rsync 源码。
#
# 用法（仓库根目录，main 分支）：
#   ./scripts/deploy-production.sh              # 部署 origin/main 上最新 tag
#   ./scripts/deploy-production.sh v1.1.0       # 部署指定 tag
# 前置条件：
#   1. tag 已推送且 CI「Release」工作流成功（Release 已挂镜像包）
#   2. tag 在 origin/main 上
# 环境变量：SERVER / REMOTE_DIR 同前；DEPLOY_DB_USER / DEPLOY_DB_NAME 覆盖备份库配置；
#   DEPLOY_DB_CONTAINER 覆盖备份用的 postgres 容器名（默认 hotspot-postgres-1，
#   依赖服务器部署目录名为 hotspot，与 scripts/backup-postgres.sh 的 CONTAINER 对应）；
#   ALERT_WEBHOOK 可选（接收 POST JSON {"text":...} 的机器人地址，同 health-check.sh），
#   版本校验失败触发自动回滚时推送告警。
# 安全网：滚动更新前自动 pg_dump（服务器 backups/ + 本机 .workwork/ 各一份），
# 备份失败即中止部署——迁移不可逆，无快照不上线。
# 自动回滚：更新前记录线上版本指纹；更新后版本校验失败自动重部署上一版本并
# 告警（仅回滚镜像，不含数据——迁移不可逆，见 docs/backup-restore.md）。
#
# 回滚（服务器侧手动）：IMAGE_TAG=<旧版本> docker compose -f docker-compose.yml \
#   -f docker-compose.prod.yml --profile prod up -d && docker restart hotspot-nginx-1
set -euo pipefail

SERVER="${SERVER:-workwork-server}"
REMOTE_DIR="${REMOTE_DIR:-~/hotspot}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGING="${REPO_ROOT}/.workwork/deploy-staging"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod"

TARGET="${1:-}"
if [ -z "$TARGET" ]; then
  TARGET="$(git -C "$REPO_ROOT" describe --tags --abbrev=0 'origin/main^{commit}')"
  echo "==> 未指定 tag，使用 origin/main 上最新 tag：${TARGET}"
fi
if ! [[ "$TARGET" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+].*)?$ ]]; then
  echo "!! 参数必须是 vX.Y.Z 形态的 tag（当前值：${TARGET}）"; exit 1
fi

echo "==> 校验 tag 与工作区…"
git -C "$REPO_ROOT" fetch origin main --quiet
git -C "$REPO_ROOT" rev-parse -q --verify "refs/tags/$TARGET" >/dev/null || { echo "!! tag ${TARGET} 不存在"; exit 1; }
# 防止把未合并代码发上线
git -C "$REPO_ROOT" merge-base --is-ancestor "$TARGET" origin/main || { echo "!! tag ${TARGET} 不在 origin/main 上"; exit 1; }
[ -z "$(git -C "$REPO_ROOT" status --porcelain)" ] || { echo "!! 工作区有未提交改动"; exit 1; }

COMMIT="$(git -C "$REPO_ROOT" rev-parse "${TARGET}^{commit}")"
echo "==> 部署版本: ${TARGET} (commit ${COMMIT:0:12})"

# docker-compose.prod.yml 的 `ports: !override []` 依赖 YAML merge tag（Compose v2.24+），
# 过旧版本解析报错晦涩，提前拦下。
MIN_COMPOSE="2.24.0"
SERVER_COMPOSE="$(ssh "$SERVER" "docker compose version --short")"
if [ "$(printf '%s\n%s\n' "$MIN_COMPOSE" "$SERVER_COMPOSE" | sort -V | head -1)" != "$MIN_COMPOSE" ]; then
  echo "!! 服务器 Docker Compose ${SERVER_COMPOSE} 过旧，需 ≥ v${MIN_COMPOSE}"; exit 1
fi
echo "    服务器 Docker Compose: ${SERVER_COMPOSE}"

echo "==> 从 GitHub Release 下载镜像包…"
mkdir -p "$STAGING"
( cd "$REPO_ROOT" && gh release download "$TARGET" --dir "$STAGING" --clobber --pattern 'hotspot-images-*.tar.gz' )
TARBALL="$STAGING/hotspot-images-$TARGET.tar.gz"
[ -f "$TARBALL" ] || { echo "!! Release ${TARGET} 缺少镜像包（CI Release 工作流是否成功？）"; exit 1; }
ls -lh "$TARBALL"

echo "==> 同步部署配置（compose + 仓库派生 nginx 片段，非源码）…"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/tmp-deploy-incoming"
scp -q "$REPO_ROOT/docker-compose.yml" \
       "$REPO_ROOT/docker-compose.prod.yml" \
       "$REPO_ROOT/docker/nginx.headers.inc" \
       "$REPO_ROOT/docker/nginx.proxy.inc" \
       "$REPO_ROOT/docker/nginx.prod-tls.conf" \
       "$REPO_ROOT/docker/prod-server.inc" \
       "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/"
ssh "$SERVER" "cd $REMOTE_DIR && mv tmp-deploy-incoming/docker-compose.yml tmp-deploy-incoming/docker-compose.prod.yml . && \
  mv tmp-deploy-incoming/nginx.headers.inc tmp-deploy-incoming/nginx.proxy.inc tmp-deploy-incoming/nginx.prod-tls.conf tmp-deploy-incoming/prod-server.inc docker/ && rmdir tmp-deploy-incoming"

echo "==> 上传镜像包并 docker load（${TARGET}）…"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/tmp-deploy-incoming"
scp -q "$TARBALL" "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz"
ssh "$SERVER" "docker load -i $REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz && \
  rm -rf $REMOTE_DIR/tmp-deploy-incoming"

# ── 部署前数据库备份（硬性步骤，失败即中止部署）──────────────────────
# 新版本迁移不可逆（无 down 迁移），上线前必须先有快照：服务器本地一份 +
# scp 回本机一份（本机副本即异地备份，服务器磁盘故障也有退路）。格式与
# backup-postgres.sh 一致用 -Fc custom format，dump 后立即 pg_restore -l
# 校验归档完整性——坏备份等同没有备份，同样中止部署。
echo "==> 部署前数据库备份…"
DEPLOY_DB_USER="${DEPLOY_DB_USER:-postgres}"
DEPLOY_DB_NAME="${DEPLOY_DB_NAME:-hotspot}"
DEPLOY_DB_CONTAINER="${DEPLOY_DB_CONTAINER:-hotspot-postgres-1}"
BACKUP_STAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_NAME="hotspot_pre_deploy_${TARGET}_${BACKUP_STAMP}.dump"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/backups && docker exec $DEPLOY_DB_CONTAINER pg_dump -U $DEPLOY_DB_USER -d $DEPLOY_DB_NAME -Fc --no-owner > $REMOTE_DIR/backups/$BACKUP_NAME"
ssh "$SERVER" "docker exec -i $DEPLOY_DB_CONTAINER pg_restore -l >/dev/null < $REMOTE_DIR/backups/$BACKUP_NAME" \
  || { echo "!! 备份完整性校验失败（pg_restore -l 非零），中止部署"; exit 1; }
scp -q "$SERVER:$REMOTE_DIR/backups/$BACKUP_NAME" "$REPO_ROOT/.workwork/$BACKUP_NAME"
BACKUP_SIZE="$(ssh "$SERVER" "du -h $REMOTE_DIR/backups/$BACKUP_NAME | cut -f1")"
echo "✅ 备份完成并已校验: ${BACKUP_NAME}（${BACKUP_SIZE}，服务器 $REMOTE_DIR/backups/ + 本机 .workwork/）"

echo "==> 读取当前线上版本（作为自动回滚目标）…"
PREVIOUS_TAG="$(ssh "$SERVER" "docker inspect hotspot-backend-1 --format '{{index .Config.Labels \"org.opencontainers.image.version\"}}'" 2>/dev/null || true)"
if [ -n "$PREVIOUS_TAG" ]; then
  echo "    当前线上版本: ${PREVIOUS_TAG}"
else
  echo "    读取失败或首次部署：本次部署无自动回滚网（校验失败将只告警不回滚）"
fi

# 告警送达（与 scripts/health-check.sh 同一约定：POST {"text": ...}）。
alert() {
  [ -n "${ALERT_WEBHOOK:-}" ] || return 0
  local escaped
  escaped="$(printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')"
  curl -sf -m 10 -H 'Content-Type: application/json' \
    -d "{\"text\": \"$escaped\"}" "$ALERT_WEBHOOK" >/dev/null 2>&1 || echo "!! ALERT_WEBHOOK 推送失败"
}

# 自动回滚：仅重部署上一版本镜像（数据不回滚——迁移不可逆，需按 runbook 评估）。
rollback() {
  echo "!! $1"
  if [ -n "$PREVIOUS_TAG" ]; then
    echo "==> 自动回滚到 ${PREVIOUS_TAG}…"
    if ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$PREVIOUS_TAG' $COMPOSE up -d && docker restart hotspot-nginx-1 >/dev/null"; then
      echo "✅ 已回滚至 ${PREVIOUS_TAG}。注意：镜像可回滚、数据迁移不可逆，若新版本已启动并执行迁移，按 docs/backup-restore.md 评估数据面。"
      alert "HotSpot 部署 ${TARGET} 校验失败，已自动回滚至 ${PREVIOUS_TAG}。原因：$1"
    else
      alert "HotSpot 部署 ${TARGET} 校验失败，且自动回滚执行出错——需人工介入！原因：$1"
    fi
  else
    alert "HotSpot 部署 ${TARGET} 校验失败（无回滚网/首次部署）——需人工介入！原因：$1"
  fi
  exit 1
}

echo "==> 滚动更新（IMAGE_TAG=${TARGET}；frontend 容器启动时自刷新静态卷）…"
ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$TARGET' $COMPOSE up -d && docker restart hotspot-nginx-1 >/dev/null && echo 'nginx restarted'"

echo "==> 部署自校验（等待 health 就绪并核对版本指纹）…"
backend_ok=""
for _ in $(seq 1 30); do
  sleep 5
  HEALTH="$(ssh "$SERVER" "docker exec hotspot-backend-1 curl -sf -X POST -H 'Content-Type: application/json' \
    -d '{}' http://localhost:8081/public/v1/system/health" 2>/dev/null || true)"
  if echo "$HEALTH" | grep -q "\"version\":\"$TARGET\""; then
    echo "✅ 后端版本指纹一致: $HEALTH"
    backend_ok=1
    break
  fi
done
[ -n "$backend_ok" ] || rollback "150s 内后端版本校验未通过（检查 docker logs hotspot-backend-1）"

frontend_ok=""
for i in $(seq 1 12); do
  FE="$(ssh "$SERVER" "docker exec hotspot-frontend-1 cat /build/version.json" 2>/dev/null || true)"
  if echo "$FE" | grep -q "\"version\": *\"$TARGET\""; then
    echo "✅ 前端版本指纹一致: $FE"
    frontend_ok=1
    break
  fi
  sleep 2
done
# 不能用循环变量 _ 判断是否走完最后一轮：_ 即 bash 特殊变量 $_，会被循环体内
# 的命令覆盖（sleep 等），导致校验失败也不中止、部署被误报成功。
[ -n "$frontend_ok" ] || rollback "前端静态卷版本校验未通过: ${FE:-<空>}"

rm -f "$TARBALL"
echo "🎉 部署完成: ${TARGET} → ${SERVER}:${REMOTE_DIR}（回滚：IMAGE_TAG=<旧版本> ${COMPOSE} up -d && docker restart hotspot-nginx-1）"
