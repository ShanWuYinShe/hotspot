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
# 环境变量：SERVER / REMOTE_DIR 同前。
#
# 回滚（服务器侧）：IMAGE_TAG=<旧版本> docker compose -f docker-compose.yml \
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
scp -q "$TARBALL" "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz"
ssh "$SERVER" "docker load -i $REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz && \
  rm -rf $REMOTE_DIR/tmp-deploy-incoming"

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
[ -n "$backend_ok" ] || { echo "!! 150s 内后端版本校验未通过，检查 docker logs hotspot-backend-1"; exit 1; }

frontend_ok=""
for _ in $(seq 1 12); do
  FE="$(ssh "$SERVER" "docker exec hotspot-frontend-1 cat /build/version.json" 2>/dev/null || true)"
  if echo "$FE" | grep -q "\"version\": *\"$TARGET\""; then
    echo "✅ 前端版本指纹一致: $FE"
    frontend_ok=1
    break
  fi
  if [ "$_" = "12" ]; then
    echo "!! 前端静态卷版本校验未通过: $FE"; exit 1
  fi
  sleep 2
done

rm -f "$TARBALL"
echo "🎉 部署完成: ${TARGET} → ${SERVER}:${REMOTE_DIR}（回滚：IMAGE_TAG=<旧版本> ${COMPOSE} up -d && docker restart hotspot-nginx-1）"
