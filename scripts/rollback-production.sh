#!/usr/bin/env bash
# ── HotSpot 生产回滚（一键，复用部署链路）───────────────────────────
#
# 用法（仓库根目录）：
#   ./scripts/rollback-production.sh --previous   # 回滚到上一次成功部署的版本
#   ./scripts/rollback-production.sh v1.1.9       # 回滚到指定版本
#   ./scripts/rollback-production.sh --list       # 只看当前版本与部署历史，不改动
#   追加 --yes 跳过确认（自动化/紧急场景）
#
# 设计：回滚 = 用旧 tag 再部署一次，因此直接复用 scripts/deploy-production.sh
# （同一套 tag 校验、发布包 sha256 校验、部署前备份、滚动更新、版本指纹核对、
# 失败自动回滚网），只额外做三件事：交互确认、审计记为 kind=rollback、完成后
# 提示数据面风险。刻意不复制部署逻辑——两条链路各自演化必然漂移。
#
# 数据面提醒：本脚本只回滚镜像。迁移不可逆（无 down 迁移），若新版本已执行
# 迁移，回滚后需按 docs/backup-restore.md 评估 schema 兼容性；紧急情况下先
# 恢复服务，再评估数据。
# 环境变量：SERVER / REMOTE_DIR 同 scripts/deploy-production.sh，其余（备份、
# 告警、镜像保留）由部署脚本按其自身默认值处理。
set -euo pipefail

SERVER="${SERVER:-workwork-server}"
REMOTE_DIR="${REMOTE_DIR:-~/hotspot}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_SCRIPT="$REPO_ROOT/scripts/deploy-production.sh"

ASSUME_YES=0
TARGET=""
for arg in "$@"; do
  case "$arg" in
    --previous) TARGET="__previous__" ;;
    --list) TARGET="__list__" ;;
    --yes|-y) ASSUME_YES=1 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    -*) echo "!! 未知参数：${arg}（可用：<vX.Y.Z> --previous --list --yes）"; exit 1 ;;
    *) TARGET="$arg" ;;
  esac
done
[ -n "$TARGET" ] || { echo "用法: $0 <vX.Y.Z|--previous|--list> [--yes]"; exit 1; }

# 当前线上版本：按 compose 服务名动态定位容器（与部署脚本同一约定）
CONTAINER="$(ssh "$SERVER" "cd $REMOTE_DIR && docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps -q backend 2>/dev/null | head -n1" || true)"
CURRENT=""
if [ -n "$CONTAINER" ]; then
  CURRENT="$(ssh "$SERVER" "docker inspect $CONTAINER --format '{{index .Config.Labels \"org.opencontainers.image.version\"}}'" 2>/dev/null || true)"
fi

HISTORY_CMD="grep 'result=success' $REMOTE_DIR/backups/deploy-history.log 2>/dev/null | tail -n 20"

if [ "$TARGET" = "__list__" ]; then
  echo "==> 当前线上版本: ${CURRENT:-<未知>}"
  echo "==> 部署历史（最近 20 条成功记录，服务器 $REMOTE_DIR/backups/deploy-history.log）："
  ssh "$SERVER" "$HISTORY_CMD" 2>/dev/null | sed 's/^/    /' || echo "    （暂无历史；该文件自本次加固后开始记录）"
  echo "==> 本地可用 tag（最近 10 个）："
  git -C "$REPO_ROOT" tag --sort=-v:refname | head -10 | sed 's/^/    /'
  exit 0
fi

if [ "$TARGET" = "__previous__" ]; then
  # 「上一次成功部署且不等于当前版本」——直接读审计日志，不靠人工回忆。
  # 当前版本未知时不敢猜（历史最新一条恰是当前版本），要求显式指定。
  if [ -z "$CURRENT" ]; then
    echo "!! 读取不到当前线上版本，无法推断上一版本；请显式指定 $0 vX.Y.Z"
    exit 1
  fi
  TARGET="$(ssh "$SERVER" "$HISTORY_CMD" 2>/dev/null | awk -F'tag=' '{split($2,a," "); print a[1]}' | grep -vx "$CURRENT" | tail -n1 || true)"
  if [ -z "$TARGET" ]; then
    echo "!! 部署历史里没有早于 $CURRENT 的成功记录（历史为空或只部署过一次）。"
    echo "   请显式指定：$0 vX.Y.Z（可用 $0 --list 查看）"
    exit 1
  fi
  echo "==> 依据部署历史推断上一版本：$TARGET"
fi

if ! [[ "$TARGET" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "!! 参数必须是 vX.Y.Z 形态的 tag（当前值：${TARGET}）"; exit 1
fi
if [ "$TARGET" = "$CURRENT" ]; then
  echo "==> 线上已经是 ${TARGET}，无需回滚"; exit 0
fi

echo ""
echo "██████ 回滚计划 ██████"
echo "  当前线上版本 : ${CURRENT:-<未知>}"
echo "  回滚目标版本 : $TARGET"
echo "  数据面       : 不回滚（迁移不可逆；如需恢复数据见 docs/backup-restore.md）"
echo "  执行方式     : $DEPLOY_SCRIPT ${TARGET}（同一套校验与自动回滚网）"
echo "  部署前会照常做一次数据库快照（备份失败即中止，不会带着无快照的状态上线）"
echo ""
if [ "$ASSUME_YES" != "1" ]; then
  printf '确认回滚？输入 yes 继续：'
  read -r answer
  [ "$answer" = "yes" ] || { echo "已取消（未做任何改动）"; exit 1; }
fi

echo "==> 开始回滚…"
ROLLBACK_OF="${CURRENT:-unknown}" "$DEPLOY_SCRIPT" "$TARGET"

echo ""
echo "🎉 回滚完成：${CURRENT:-<未知>} → $TARGET"
echo "   若线上出现数据相关的报错，按 docs/backup-restore.md 处理（镜像已回退，schema 不会自动回退）"