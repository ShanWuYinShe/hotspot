#!/usr/bin/env bash
# ── HotSpot 生产部署脚本（本地执行，SSH 到生产服务器）──────────────────
#
# 把 git 版本号贯穿整条发布链路，保证线上出问题时可精确回溯：
#   git describe → IMAGE_TAG（镜像 tag，docker images 可见）
#               → GIT_VERSION/GIT_COMMIT/GIT_BUILD_TIME（注入两镜像）
#               → 启动日志 + /public/v1/system/health + /version.json
#
# 用法（仓库根目录，main 分支）：
#   ./scripts/deploy-production.sh                # 部署 origin/main HEAD
#   ./scripts/deploy-production.sh v1.0.1         # 部署指定 tag
# 环境变量：
#   SERVER=workwork-server REMOTE_DIR=~/hotspot   # SSH 目标与远端目录
#   SKIP_BUILD=1                                  # 跳过镜像构建（仅同步+up）
#
# 依赖：rsync、ssh；服务器侧维护文件（cert/.env/nginx.prod-tls.conf 等）
# 必须在 RSYNC_EXCLUDES 中排除——漏排会被 --delete 删除（2026-09-13 事故）。
set -euo pipefail

SERVER="${SERVER:-workwork-server}"
REMOTE_DIR="${REMOTE_DIR:-~/hotspot}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod"

TARGET="${1:-main}"
if [ "$TARGET" != "main" ] && [[ ! "$TARGET" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "!! 参数必须是 main 或 vX.Y.Z tag（当前值：${TARGET}）"; exit 1
fi

echo "==> 校验本地 main 与 origin/main 同步、工作区干净…"
git -C "$REPO_ROOT" fetch origin main --quiet
[ -z "$(git -C "$REPO_ROOT" status --porcelain)" ] || {
  echo "!! 工作区有未提交改动，先提交或清理"; exit 1; }

if [ "$TARGET" = "main" ]; then
  LOCAL="$(git -C "$REPO_ROOT" rev-parse main)"
  REMOTE="$(git -C "$REPO_ROOT" rev-parse origin/main)"
  [ "$LOCAL" = "$REMOTE" ] || { echo "!! 本地 main != origin/main，先 git pull --ff-only"; exit 1; }
  REF="origin/main"
else
  git -C "$REPO_ROOT" rev-parse -q --verify "refs/tags/$TARGET" >/dev/null || {
    echo "!! tag $TARGET 不存在"; exit 1; }
  # 部署的 tag 必须已在 origin/main 上，防止把未合并代码发上线
  git -C "$REPO_ROOT" merge-base --is-ancestor "$TARGET" origin/main || {
    echo "!! tag $TARGET 不在 origin/main 上"; exit 1; }
  REF="$TARGET"
fi

VERSION="$(git -C "$REPO_ROOT" describe --tags --always "$REF")"
COMMIT="$(git -C "$REPO_ROOT" rev-parse "$REF^{commit}")"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "==> 部署版本: $VERSION (commit ${COMMIT:0:12}, built $BUILD_TIME)"

echo "==> rsync 同步代码（排除服务器侧维护文件）…"
RSYNC_EXCLUDES=(
  --exclude .git --exclude .env --exclude backups --exclude cert
  --exclude '.workwork/' --exclude '.workbuddy/' --exclude '.worktrees/'
  --exclude 'harmonyapp/'
  --exclude '.github/' --exclude '.husky/' --exclude '.hvigor/' --exclude '.v2c/'
  --exclude 'CLAUDE.md' --exclude 'AGENTS.md' --exclude '.DS_Store'
  --exclude 'node_modules/' --exclude 'frontend/build/' --exclude 'frontend/.svelte-kit/'
  --exclude 'frontend/static/version.json'
  --exclude 'docker/amap.prod.inc' --exclude 'docker/nginx.prod-tls.conf'
  --exclude 'docker/prod-server.inc' --exclude 'docker-compose.prod.yml'
)
rsync -az --delete "${RSYNC_EXCLUDES[@]}" "$REPO_ROOT/" "$SERVER:$REMOTE_DIR/"

SSH="ssh $SERVER"
if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "==> 构建镜像（GIT_* 注入版本指纹，IMAGE_TAG=${VERSION}）…"
  $SSH "cd $REMOTE_DIR && GIT_VERSION='$VERSION' GIT_COMMIT='$COMMIT' GIT_BUILD_TIME='$BUILD_TIME' \
    IMAGE_TAG='$VERSION' $COMPOSE build backend frontend-build"
fi

echo "==> 滚动更新（IMAGE_TAG=${VERSION}，compose 比对新 tag 镜像重建容器）…"
$SSH "cd $REMOTE_DIR && IMAGE_TAG='$VERSION' $COMPOSE up -d"

# 前端静态卷刷新：Docker 命名卷非空时不会自动复制新镜像内容（2026-09-15 实测），
# 必须清空并从新镜像 tar 拷入，然后 restart nginx（upstream IP 缓存随之刷新）。
echo "==> 刷新前端静态卷 + 重启 nginx…"
$SSH "cd $REMOTE_DIR && docker rm -f fe-staging fe-fix 2>/dev/null; \
  docker run -d --name fe-staging --entrypoint sh hotspot-frontend:$VERSION -c 'sleep 300' >/dev/null && \
  docker run -d --name fe-fix -v hotspot_frontend_build:/mnt/vol --entrypoint sh hotspot-frontend:$VERSION -c 'sleep 300' >/dev/null && \
  docker exec fe-fix sh -c 'rm -rf /mnt/vol/* /mnt/vol/.[!.]* /mnt/vol/..?*' && \
  docker exec fe-staging tar -C /build -cf - . | docker exec -i fe-fix tar -C /mnt/vol -xf - && \
  docker rm -f fe-staging fe-fix >/dev/null && \
  docker restart hotspot-nginx-1 >/dev/null && echo 'frontend volume refreshed'"

echo "==> 部署自校验（等待 health 就绪并核对版本指纹）…"
ok=""
for _ in $(seq 1 30); do
  sleep 5
  HEALTH="$($SSH "docker exec hotspot-backend-1 curl -sf -X POST -H 'Content-Type: application/json' \
    -d '{}' http://localhost:8081/public/v1/system/health" 2>/dev/null || true)"
  if echo "$HEALTH" | grep -q "\"version\":\"$VERSION\""; then
    echo "✅ 后端版本指纹一致: $HEALTH"
    ok="backend"
    break
  fi
done
[ -n "$ok" ] || { echo "!! 150s 内后端版本校验未通过，检查 docker logs hotspot-backend-1"; exit 1; }

for _ in $(seq 1 10); do
  FE="$($SSH "docker exec hotspot-frontend-1 cat /build/version.json" 2>/dev/null || true)"
  if echo "$FE" | grep -q "\"version\": *\"$VERSION\""; then
    echo "✅ 前端版本指纹一致: $FE"
    break
  fi
  if [ "${_}" = "10" ]; then
    echo "!! 前端 volume 版本校验未通过: $FE"; exit 1
  fi
  sleep 2
done

echo "🎉 部署完成: ${VERSION} → ${SERVER}:${REMOTE_DIR}（镜像 tag=${VERSION}，回滚：IMAGE_TAG=<旧版本> ${COMPOSE} up -d）"
