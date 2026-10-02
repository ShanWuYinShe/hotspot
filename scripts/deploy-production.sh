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
#   DEPLOY_KEEP_IMAGES 服务器旧镜像保留个数（默认 3，0 = 不清理）；
#   DEPLOY_LOCK_TTL_HOURS 并发锁陈旧判定小时数（默认 6）；
#   ROLLBACK_OF=<被回滚掉的原版本> 由 scripts/rollback-production.sh 传入，只影响
#   审计记录与告警文案；ALERT_WEBHOOK 可选（接收 POST JSON {"text":...} 的机器人
#   地址，同 health-check.sh），版本校验失败触发自动回滚时推送告警。
#
# 同源：镜像取自 tag 对应的 Release 包，部署配置（compose + nginx 片段 + 采集配置）
# 也从同一 tag 用 git archive 导出——不看工作树当前内容，保证「线上跑的就是那个
# tag」；本地 HEAD 领先被部署 tag 时也不会把未发布的配置带上线。
# 完整性：Release 带 <包名>.sha256 时强制校验（旧版本无该资产则告警放行）；load 后
# 先核对镜像 OCI 标签 version/revision 与 tag/commit 一致，再动线上容器。
# 安全网：滚动更新前自动 pg_dump（服务器 backups/ + 本机 .workwork/ 各一份），
# 备份失败即中止部署——迁移不可逆，无快照不上线。
# 自动回滚：更新前记录线上版本指纹；更新后版本校验失败自动重部署上一版本并
# 告警（仅回滚镜像，不含数据——迁移不可逆，见 docs/backup-restore.md）。
# 审计：每次部署追加一行到服务器 backups/deploy-history.log 与本机
# .workwork/deploy-history.log（tag / commit / 上一版本 / 结果 / 操作者）。
#
# 回滚（推荐）：./scripts/rollback-production.sh [<旧版本>|--previous]——复用本脚本
#   部署旧 tag，结束后同样核对线上版本指纹。手工等价命令：
#   IMAGE_TAG=<旧版本> docker compose -f docker-compose.yml -f docker-compose.prod.yml \
#     --profile prod up -d && docker restart hotspot-nginx-1
# 完整发布/回滚/热修复流程见 docs/release-runbook.md。
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

# ── 审计与并发锁（任何变更之前）──────────────────────────────────────
HISTORY_LOCAL="$REPO_ROOT/.workwork/deploy-history.log"
DEPLOY_KIND="deploy"
[ -n "${ROLLBACK_OF:-}" ] && DEPLOY_KIND="rollback"

# 历史记录写不进去只告警不阻塞：审计是副产物，不该把一次成功部署判成失败。
record_history() {
  local result="$1" note="${2:-}" line
  line="$(date -u +%FT%TZ) kind=${DEPLOY_KIND} tag=${TARGET} commit=${COMMIT:0:12} previous=${PREVIOUS_TAG:-none} result=${result} operator=$(whoami)@$(hostname) ${note}"
  mkdir -p "$(dirname "$HISTORY_LOCAL")"
  printf '%s\n' "$line" >> "$HISTORY_LOCAL"
  ssh "$SERVER" "mkdir -p $REMOTE_DIR/backups && printf '%s\n' '$line' >> $REMOTE_DIR/backups/deploy-history.log" >/dev/null 2>&1 \
    || echo "!! 服务器部署历史写入失败（本地已记录：${HISTORY_LOCAL}）"
}

# 服务器侧原子 mkdir 互斥：两个终端/两台机器同时部署会交错 load 与 up，结果
# 无法预期。上次异常中断留下的锁超过 TTL 视为陈旧，显式接管并留痕。
LOCK_DIR="$REMOTE_DIR/.deploy.lock"
LOCK_HELD=0
LOCK_TTL_HOURS="${DEPLOY_LOCK_TTL_HOURS:-6}"
release_lock() {
  if [ "$LOCK_HELD" = "1" ]; then
    ssh "$SERVER" "rm -rf $LOCK_DIR" >/dev/null 2>&1 || true
    LOCK_HELD=0
  fi
}
acquire_lock() {
  if ssh "$SERVER" "mkdir $LOCK_DIR 2>/dev/null"; then
    ssh "$SERVER" "printf 'tag=%s\noperator=%s@%s\nstarted_at=%s\n' '$TARGET' \"\$(whoami)\" \"\$(hostname)\" \"\$(date -u +%FT%TZ)\" > $LOCK_DIR/info" >/dev/null 2>&1 || true
    LOCK_HELD=1
    trap release_lock EXIT
    echo "==> 已获取部署锁（${LOCK_DIR}）"
    return 0
  fi
  local age_h info
  age_h="$(ssh "$SERVER" "echo \$(( ( \$(date +%s) - \$(stat -c %Y $LOCK_DIR 2>/dev/null || echo 0) ) / 3600 ))" 2>/dev/null || echo 0)"
  info="$(ssh "$SERVER" "cat $LOCK_DIR/info 2>/dev/null || true")"
  if [ "${age_h:-0}" -ge "$LOCK_TTL_HOURS" ]; then
    echo "!! 检测到陈旧部署锁（已持有 ${age_h}h ≥ TTL ${LOCK_TTL_HOURS}h），接管："
    printf '%s\n' "$info" | sed 's/^/    /'
    ssh "$SERVER" "rm -rf $LOCK_DIR && mkdir $LOCK_DIR"
    LOCK_HELD=1
    trap release_lock EXIT
    return 0
  fi
  echo "!! 另一个部署正在进行（锁 ${LOCK_DIR}，已持有 ${age_h}h）："
  printf '%s\n' "$info" | sed 's/^/    /'
  echo "   确认对方已中断后执行 ssh $SERVER 'rm -rf $LOCK_DIR' 再重试。"
  exit 1
}
acquire_lock

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
( cd "$REPO_ROOT" && gh release download "$TARGET" --dir "$STAGING" --clobber \
    --pattern 'hotspot-images-*.tar.gz' --pattern 'hotspot-images-*.tar.gz.sha256' )
TARBALL="$STAGING/hotspot-images-$TARGET.tar.gz"
[ -f "$TARBALL" ] || { echo "!! Release ${TARGET} 缺少镜像包（CI Release 工作流是否成功？）"; exit 1; }
ls -lh "$TARBALL"

# 完整性校验：CI 为发布包生成同名 .sha256 资产。旧版本 Release 没有该资产时
# 只告警放行（否则无法回滚到历史版本）；有资产但校验不过一律中止——损坏或
# 被替换的镜像包上线，后果比中断一次部署严重得多。
SHA_FILE="$STAGING/hotspot-images-$TARGET.tar.gz.sha256"
if [ -f "$SHA_FILE" ]; then
  if ( cd "$STAGING" && { shasum -a 256 -c "hotspot-images-$TARGET.tar.gz.sha256" >/dev/null 2>&1 \
      || sha256sum -c "hotspot-images-$TARGET.tar.gz.sha256" >/dev/null 2>&1; } ); then
    echo "==> 镜像包 sha256 校验通过"
  else
    echo "!! 镜像包 sha256 校验失败（下载损坏或资产被替换），中止部署"; exit 1
  fi
else
  echo "!! 注意：Release ${TARGET} 未提供 .sha256 资产（多为旧版本发布包），跳过完整性校验"
fi

# ── 部署配置与镜像同源 ──────────────────────────────────────────────
# 关键：配置从被部署的 tag 导出（git archive），而不是取工作树当前文件。
# 工作树 HEAD 常常领先被部署的 tag（例如回滚到旧版本时），直接 scp 会把尚未
# 发布的 nginx/compose 配置带上线——线上就成了「新配置 + 旧镜像」的混合体，
# 现象难查且不可复现。
echo "==> 从 tag ${TARGET} 导出部署配置（与镜像同源）…"
CONFIG_STAGING="$STAGING/config"
rm -rf "$CONFIG_STAGING"
mkdir -p "$CONFIG_STAGING"
CONFIG_PATHS="docker-compose.yml docker-compose.prod.yml docker/nginx.headers.inc docker/nginx.proxy.inc docker/nginx.prod-tls.conf docker/prod-server.inc docker/prometheus.yml docker/grafana-provisioning"
ARCHIVE_PATHS=""
for p in $CONFIG_PATHS; do
  if git -C "$REPO_ROOT" cat-file -e "${TARGET}:${p}" 2>/dev/null; then
    ARCHIVE_PATHS="$ARCHIVE_PATHS $p"
  else
    echo "    注意：tag ${TARGET} 中不存在 ${p}，跳过（旧版本可能尚未引入该文件）"
  fi
done
# shellcheck disable=SC2086  # 路径列表按空格拆分传给 git archive
git -C "$REPO_ROOT" archive "$TARGET" $ARCHIVE_PATHS | tar -x -C "$CONFIG_STAGING"

echo "==> 同步部署配置（compose + nginx 片段 + observability 采集配置，非源码）…"
SCP_LIST="$CONFIG_STAGING/docker-compose.yml $CONFIG_STAGING/docker-compose.prod.yml"
for f in nginx.headers.inc nginx.proxy.inc nginx.prod-tls.conf prod-server.inc prometheus.yml; do
  [ -f "$CONFIG_STAGING/docker/$f" ] && SCP_LIST="$SCP_LIST $CONFIG_STAGING/docker/$f"
done
[ -d "$CONFIG_STAGING/docker/grafana-provisioning" ] && SCP_LIST="$SCP_LIST $CONFIG_STAGING/docker/grafana-provisioning"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/tmp-deploy-incoming"
# shellcheck disable=SC2086  # 同上
scp -q -r $SCP_LIST "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/"

REMOTE_MOVE="cd $REMOTE_DIR && mkdir -p docker && mv tmp-deploy-incoming/docker-compose.yml tmp-deploy-incoming/docker-compose.prod.yml ."
for f in nginx.headers.inc nginx.proxy.inc nginx.prod-tls.conf prod-server.inc prometheus.yml; do
  if [ -f "$CONFIG_STAGING/docker/$f" ]; then
    REMOTE_MOVE="$REMOTE_MOVE && mv tmp-deploy-incoming/$f docker/$f"
  fi
done
if [ -d "$CONFIG_STAGING/docker/grafana-provisioning" ]; then
  REMOTE_MOVE="$REMOTE_MOVE && rm -rf docker/grafana-provisioning && mv tmp-deploy-incoming/grafana-provisioning docker/grafana-provisioning"
fi
REMOTE_MOVE="$REMOTE_MOVE && rmdir tmp-deploy-incoming"
ssh "$SERVER" "$REMOTE_MOVE"
# observability 分发说明：prometheus.yml 与 grafana-provisioning/ 是 --profile
# observability 的 bind-mount 源，缺失会导致 up 时 prometheus/grafana 起不来，
# 故随部署配置一并分发（grafana-provisioning 整目录替换，服务器侧无本地定制）。

echo "==> 上传镜像包并 docker load（${TARGET}）…"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/tmp-deploy-incoming"
scp -q "$TARBALL" "$SERVER:$REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz"
ssh "$SERVER" "docker load -i $REMOTE_DIR/tmp-deploy-incoming/hotspot-images-$TARGET.tar.gz && \
  rm -rf $REMOTE_DIR/tmp-deploy-incoming"

# 加载后立刻核对 OCI 标签：version 必须等于 tag、revision 必须等于该 tag 的 commit。
# 这一步挡住「发布包与 tag 不一致」（资产被替换、CI 打错 commit）——版本漂移在
# 动线上容器之前暴露，比事后回滚便宜得多。
echo "==> 核对已加载镜像的版本指纹（镜像 ↔ tag commit）…"
for repo in hotspot-backend hotspot-frontend; do
  ver="$(ssh "$SERVER" "docker inspect ${repo}:${TARGET} --format '{{index .Config.Labels \"org.opencontainers.image.version\"}}'" 2>/dev/null || true)"
  rev="$(ssh "$SERVER" "docker inspect ${repo}:${TARGET} --format '{{index .Config.Labels \"org.opencontainers.image.revision\"}}'" 2>/dev/null || true)"
  [ "$ver" = "$TARGET" ] || { echo "!! ${repo}:${TARGET} 的 version 标签为 ${ver:-<空>}，与 tag 不符，中止部署"; exit 1; }
  [ "$rev" = "$COMMIT" ] || { echo "!! ${repo}:${TARGET} 的 revision 标签为 ${rev:-<空>}，与 ${TARGET} 的 commit ${COMMIT:0:12} 不符（发布包与 tag 不一致），中止部署"; exit 1; }
  echo "    ${repo}: version=${ver} revision=${rev:0:12}"
done

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
  record_history failed "$1"
  if [ -n "$PREVIOUS_TAG" ]; then
    echo "==> 自动回滚到 ${PREVIOUS_TAG}…"
    # --scale frontend-build=0：frontend-build 只是 build target（复用 frontend
    # 镜像、CMD 以 sleep infinity 保活），被 up -d 连带拉起后会在生产常驻一个
    # 无作用容器（线上实测 Up 23h），显式缩容到 0。
    if ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$PREVIOUS_TAG' $COMPOSE up -d --scale frontend-build=0 && docker restart $NGINX_CONTAINER >/dev/null"; then
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
ssh "$SERVER" "cd $REMOTE_DIR && IMAGE_TAG='$TARGET' $COMPOSE up -d --scale frontend-build=0 && docker restart $NGINX_CONTAINER >/dev/null && echo 'nginx restarted'"

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
# 不能用循环变量 _ 判断是否走完最后一轮：_ 即 bash 特殊变量 ${_}，会被循环体内
# 的命令覆盖（sleep 等），导致校验失败也不中止、部署被误报成功。
[ -n "$frontend_ok" ] || rollback "前端静态卷版本校验未通过: ${FE:-<空>}"

# 容器里的文件对 ≠ 网关服务得出去：卷属主/权限错误、nginx 指错目录、配置未生效时，
# 前两项校验照样全绿而站点仍 404/白屏。这里从 nginx 容器内部取一次真实响应，
# 顺带验证「网关 → 后端」链路（/healthz 由 nginx 反代到 backend）。
echo "==> 网关侧校验（nginx 实际对外提供的静态产物与存活探针）…"
gateway_ok=""
for i in $(seq 1 10); do
  GW_VERSION="$(ssh "$SERVER" "docker exec $NGINX_CONTAINER wget -qO- http://127.0.0.1/version.json" 2>/dev/null || true)"
  GW_HEALTH="$(ssh "$SERVER" "docker exec $NGINX_CONTAINER wget -qO- http://127.0.0.1/healthz" 2>/dev/null || true)"
  if echo "$GW_VERSION" | grep -q "\"version\": *\"$TARGET\"" && echo "$GW_HEALTH" | grep -q '"status":"ok"'; then
    echo "✅ 网关校验通过: $(echo "$GW_VERSION" | tr -d '\n')"
    gateway_ok=1
    break
  fi
  sleep 3
done
[ -n "$gateway_ok" ] || rollback "网关侧校验未通过（version.json 或 /healthz 响应异常: ver=${GW_VERSION:-<空>} health=${GW_HEALTH:-<空>}）"

record_history success
rm -f "$TARBALL"

# ── 旧镜像回收（默认保留 3 个历史版本）──────────────────────────────
# 每次发布都会在服务器留下两个镜像（前后端），不回收会单调增长直到磁盘告警。
# 只删 hotspot-backend/frontend 的 v* tag，且显式跳过当前版本与上一版本
# （回滚网需要上一版本镜像仍在）。
KEEP_IMAGES="${DEPLOY_KEEP_IMAGES:-3}"
if [ "$KEEP_IMAGES" -gt 0 ] 2>/dev/null; then
  echo "==> 回收服务器旧版本镜像（保留最近 ${KEEP_IMAGES} 个历史版本 + 当前/上一版本）…"
  ssh "$SERVER" "sh -s" <<REMOTE_PRUNE || echo "!! 旧镜像清理失败（不影响本次部署结果）"
keep=$KEEP_IMAGES; cur='$TARGET'; prev='${PREVIOUS_TAG:-}'
for repo in hotspot-backend hotspot-frontend; do
  docker images --format '{{.Repository}}:{{.Tag}}' "\$repo" \
    | grep -E "^\$repo:v[0-9]" \
    | grep -v -E ":(\$cur|\$prev)\$" \
    | sort -Vr | tail -n +\$((keep+1)) \
    | xargs -r docker rmi >/dev/null 2>&1 || true
done
docker image prune -f >/dev/null 2>&1 || true
REMOTE_PRUNE
fi

echo "🎉 部署完成: ${TARGET} → ${SERVER}:${REMOTE_DIR}"
echo "   回滚：./scripts/rollback-production.sh --previous（或指定 <版本>）"
echo "   历史：ssh $SERVER 'tail -5 $REMOTE_DIR/backups/deploy-history.log'"
