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
#   DEPLOY_DB_CONTAINER 覆盖备份用的 postgres 容器名（默认按 compose 服务动态探测
#   运行中容器，探测不到时回退 hotspot-postgres-1，与 scripts/backup-postgres.sh
#   的 CONTAINER 对应）；DEPLOY_FRONTEND_VOLUME 覆盖前端静态卷名（默认
#   hotspot_frontend_build，更新前会幂等迁移卷属主，见下文非 root 说明）；
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

# ── 部署前弱口令预检（告警不阻塞）───────────────────────────────────
# 生产 .env 若仍保留 docker-compose.yml 的默认/示例弱值，等于把数据库、
# 对象存储与会话签名密钥暴露给任何能触达端口的人（数据库拖库、上传接口被
# 白嫖、token 可被任意伪造——后者等于整个认证体系失效）。此处只告警不硬
# 阻塞：脚本无法确认服务器 .env 的实际防护现状（例如已由防火墙/内网隔离
# 兜底、或正处于无法改配置的紧急修复窗口），硬失败会阻断下次部署；但警告
# 必须足够醒目，并留 5 秒给操作者 Ctrl-C。
# 校验对照 compose 各服务默认值：POSTGRES_PASSWORD/S3_SECRET_KEY 默认
# change-me，S3_ACCESS_KEY 默认 minioadmin；AUTH_TOKEN_SECRET 虽为 compose
# 强制变量，但 .env 里可能填了 dev 示例值或过短的高熵不足值。
DEPLOY_WEAK_ENV_WARNINGS=""
weak_env_check() {
  local envs name val reason
  envs="$(ssh "$SERVER" "cat $REMOTE_DIR/.env 2>/dev/null || true")"
  if [ -z "$envs" ]; then
    echo "==> 未读到服务器 ${REMOTE_DIR}/.env，跳过弱口令预检"
    return 0
  fi
  # 从 .env 文本提取变量值：取首个匹配行，去掉行内注释与首尾引号（不回显值本身）
  env_val() {
    printf '%s\n' "$envs" | grep -E "^$1=" | head -n1 | cut -d= -f2- \
      | sed -E 's/[[:space:]]*#.*$//; s/^["'\'']//; s/["'\'']$//'
  }
  for name in POSTGRES_PASSWORD S3_SECRET_KEY AUTH_TOKEN_SECRET; do
    val="$(env_val "$name")"
    reason=""
    case "$name" in
      POSTGRES_PASSWORD|S3_SECRET_KEY)
        if [ -z "$val" ]; then reason="为空"
        elif [ "$val" = "change-me" ] || [ "$val" = "minioadmin" ]; then reason="仍为 compose 默认值"
        fi
        ;;
      AUTH_TOKEN_SECRET)
        if [ -z "$val" ]; then reason="为空（compose 将直接拒绝启动）"
        elif [ "$val" = "change-me" ] || [ "$val" = "minioadmin" ] \
          || [ "$val" = "dev-secret-change-me" ] \
          || [ "$val" = "dev-only-do-not-use-in-production-please-change-to-random-48-bytes" ]; then
          reason="仍为 dev/compose 示例值"
        elif [ "${#val}" -lt 32 ]; then reason="长度不足 32 字节，熵不足易被暴力破解"
        fi
        ;;
    esac
    if [ -n "$reason" ]; then
      DEPLOY_WEAK_ENV_WARNINGS="${DEPLOY_WEAK_ENV_WARNINGS}  - ${name}：${reason}\n"
    fi
  done
  if [ -n "$DEPLOY_WEAK_ENV_WARNINGS" ]; then
    echo ""
    echo "██████████████████████████████████████████████████████████████████████"
    echo "██  ⚠️  弱口令 / 默认密钥告警（服务器 ${REMOTE_DIR}/.env）  ⚠️            ██"
    echo "██████████████████████████████████████████████████████████████████████"
    printf '%b' "$DEPLOY_WEAK_ENV_WARNINGS"
    echo "  风险：数据库可被任意连接拖库、对象存储可被匿名读写、认证 token 可被"
    echo "        伪造（等于整个登录体系失效）。"
    echo "  指引：在服务器 ${REMOTE_DIR}/.env 中为上述变量填入强随机值"
    echo "        （openssl rand -base64 48 生成）。注意：POSTGRES_PASSWORD 仅在"
    echo "        数据卷首次初始化时生效，已有数据需 ALTER USER 同步；更换"
    echo "        AUTH_TOKEN_SECRET 会使所有已登录会话失效（需重新登录）；"
    echo "        S3_* 改后 docker compose up -d 重建 seaweedfs 与 backend 生效。"
    echo "  5 秒后继续部署，按 Ctrl-C 中止并先修改 .env …"
    sleep 5
  else
    echo "==> 弱口令预检通过（POSTGRES_PASSWORD / S3_SECRET_KEY / AUTH_TOKEN_SECRET）"
  fi
}
weak_env_check

# 按服务名动态探测线上容器名：compose 默认容器名 <项目>-<服务>-<序号> 依赖
# 部署目录名为 hotspot，目录改名或编号变化时硬编码即失效。ps -q 只列运行中
# 容器；探测不到（首次部署/服务未启动）时回退旧默认命名，行为与从前等价。
container_of() {
  local svc="$1" cid
  cid="$(ssh "$SERVER" "cd $REMOTE_DIR && $COMPOSE ps -q $svc 2>/dev/null" | head -n1)"
  if [ -n "$cid" ]; then
    ssh "$SERVER" "docker inspect --format '{{.Name}}' $cid" | sed 's#^/##'
  else
    echo "hotspot-${svc}-1"
  fi
}
echo "==> 探测线上容器名…"
DB_CONTAINER="${DEPLOY_DB_CONTAINER:-$(container_of postgres)}"
BACKEND_CONTAINER="$(container_of backend)"
FRONTEND_CONTAINER="$(container_of frontend)"
NGINX_CONTAINER="$(container_of nginx)"
echo "    postgres=${DB_CONTAINER} backend=${BACKEND_CONTAINER} frontend=${FRONTEND_CONTAINER} nginx=${NGINX_CONTAINER}"

echo "==> 从 GitHub Release 下载镜像包…"
mkdir -p "$STAGING"
( cd "$REPO_ROOT" && gh release download "$TARGET" --dir "$STAGING" --clobber --pattern 'hotspot-images-*.tar.gz' )
TARBALL="$STAGING/hotspot-images-$TARGET.tar.gz"
[ -f "$TARBALL" ] || { echo "!! Release ${TARGET} 缺少镜像包（CI Release 工作流是否成功？）"; exit 1; }
ls -lh "$TARBALL"

echo "==> 同步部署配置（compose + 仓库派生 nginx 片段 + observability 采集配置，非源码）…"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/tmp-deploy-incoming"
scp -q -r "$REPO_ROOT/docker-compose.yml" \
       "$REPO_ROOT/docker-compose.prod.yml" \
       "$REPO_ROOT/docker/nginx.headers.inc" \
       "$REPO_ROOT/docker/nginx.proxy.inc" \
       "$REPO_ROOT/docker/nginx.prod-tls.conf" \
       "$REPO_ROOT/docker/prod-server.inc" \
       "$REPO_ROOT/docker/prometheus.yml" \
       "$REPO_ROOT/docker/grafana-provisioning" \
       "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/"
ssh "$SERVER" "cd $REMOTE_DIR && mv tmp-deploy-incoming/docker-compose.yml tmp-deploy-incoming/docker-compose.prod.yml . && \
  mv tmp-deploy-incoming/nginx.headers.inc tmp-deploy-incoming/nginx.proxy.inc tmp-deploy-incoming/nginx.prod-tls.conf tmp-deploy-incoming/prod-server.inc tmp-deploy-incoming/prometheus.yml docker/ && \
  rm -rf docker/grafana-provisioning && mv tmp-deploy-incoming/grafana-provisioning docker/grafana-provisioning && rmdir tmp-deploy-incoming"
# observability 分发说明：prometheus.yml 与 grafana-provisioning/ 是 --profile
# observability 的 bind-mount 源，缺失会导致 up 时 prometheus/grafana 起不来，
# 故随部署配置一并分发（grafana-provisioning 整目录替换，服务器侧无本地定制）。

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
BACKUP_STAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_NAME="hotspot_pre_deploy_${TARGET}_${BACKUP_STAMP}.dump"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/backups && docker exec $DB_CONTAINER pg_dump -U $DEPLOY_DB_USER -d $DEPLOY_DB_NAME -Fc --no-owner > $REMOTE_DIR/backups/$BACKUP_NAME"
ssh "$SERVER" "docker exec -i $DB_CONTAINER pg_restore -l >/dev/null < $REMOTE_DIR/backups/$BACKUP_NAME" \
  || { echo "!! 备份完整性校验失败（pg_restore -l 非零），中止部署"; exit 1; }
scp -q "$SERVER:$REMOTE_DIR/backups/$BACKUP_NAME" "$REPO_ROOT/.workwork/$BACKUP_NAME"
BACKUP_SIZE="$(ssh "$SERVER" "du -h $REMOTE_DIR/backups/$BACKUP_NAME | cut -f1")"
echo "✅ 备份完成并已校验: ${BACKUP_NAME}（${BACKUP_SIZE}，服务器 $REMOTE_DIR/backups/ + 本机 .workwork/）"

echo "==> 读取当前线上版本（作为自动回滚目标）…"
PREVIOUS_TAG="$(ssh "$SERVER" "docker inspect $BACKEND_CONTAINER --format '{{index .Config.Labels \"org.opencontainers.image.version\"}}'" 2>/dev/null || true)"
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
    if ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$PREVIOUS_TAG' $COMPOSE up -d && docker restart $NGINX_CONTAINER >/dev/null"; then
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

# frontend 容器已非 root 运行（frontend/Dockerfile USER app，uid/gid 固定
# 100:101）：存量静态卷属主仍是 root，新容器无法清空/写入卷，更新前幂等
# 迁移一次；卷不存在时 docker run 会自动建空卷再 chown，同样安全。失败显式
# 中止——带着不可写卷滚动更新必然触发版本校验失败与回滚，不如提前拦下。
FRONTEND_VOLUME="${DEPLOY_FRONTEND_VOLUME:-hotspot_frontend_build}"
echo "==> 迁移 frontend 静态卷属主为 100:101（幂等）…"
ssh "$SERVER" "docker run --rm --user 0:0 --entrypoint sh -v $FRONTEND_VOLUME:/build hotspot-backend:$TARGET -c 'chown -R 100:101 /build'" \
  || { echo "!! frontend 静态卷（${FRONTEND_VOLUME}）属主迁移失败，中止部署"; exit 1; }

echo "==> 滚动更新（IMAGE_TAG=${TARGET}；frontend 容器启动时自刷新静态卷）…"
ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$TARGET' $COMPOSE up -d && docker restart $NGINX_CONTAINER >/dev/null && echo 'nginx restarted'"

echo "==> 部署自校验（等待 health 就绪并核对版本指纹）…"
backend_ok=""
for _ in $(seq 1 30); do
  sleep 5
  HEALTH="$(ssh "$SERVER" "docker exec $BACKEND_CONTAINER curl -sf -X POST -H 'Content-Type: application/json' \
    -d '{}' http://localhost:8081/public/v1/system/health" 2>/dev/null || true)"
  if echo "$HEALTH" | grep -q "\"version\":\"$TARGET\""; then
    echo "✅ 后端版本指纹一致: $HEALTH"
    backend_ok=1
    break
  fi
done
[ -n "$backend_ok" ] || rollback "150s 内后端版本校验未通过（检查 docker logs ${BACKEND_CONTAINER}）"

frontend_ok=""
for i in $(seq 1 12); do
  FE="$(ssh "$SERVER" "docker exec $FRONTEND_CONTAINER cat /build/version.json" 2>/dev/null || true)"
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
echo "🎉 部署完成: ${TARGET} → ${SERVER}:${REMOTE_DIR}（回滚：IMAGE_TAG=<旧版本> ${COMPOSE} up -d && docker restart ${NGINX_CONTAINER}）"
