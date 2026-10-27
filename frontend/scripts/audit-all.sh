#!/usr/bin/env bash
# UI 质量门禁一键入口：browser-qa --mobile + a11y + touch。
# 任一子检查失败 → 整体退出 1；末尾打印各子报告路径与摘要。
#
# 用法：bun run audit:ui
#       bun run audit:ui -- --base http://localhost:8898
# 前置：dev 环境已启动；agent-browser 已安装；frontend 已 bun install。

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE="http://localhost:8888"
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="${2:-$BASE}"; shift ;;
    --help|-h) sed -n '2,8p' "$0" | sed 's/^# *//'; exit 0 ;;
    *) echo "未知参数: $1（支持 --base <url>）"; exit 2 ;;
  esac
  shift
done

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

declare -a NAMES=("browser-qa" "audit:a11y" "audit:touch")
declare -a CMDS=(
  "bash $SCRIPT_DIR/browser-qa.sh --base $BASE --mobile"
  "bash $SCRIPT_DIR/browser-a11y.sh --base $BASE"
  "bash $SCRIPT_DIR/browser-touch.sh --base $BASE"
)
declare -a RCS=()
declare -a OUTS=()

for i in "${!NAMES[@]}"; do
  name="${NAMES[$i]}"
  echo ""
  echo "==================== $name ===================="
  set +e
  ${CMDS[$i]} 2>&1 | tee "$TMP_DIR/$i.log"
  rc=${PIPESTATUS[0]}
  set -e
  RCS+=("$rc")
  outs="$(awk -F'：' '/^(完成|输出)：/{print $2}' "$TMP_DIR/$i.log" | tail -1)"
  OUTS+=("${outs:-（未解析到输出目录，见上方日志）}")
  echo "---- $name 退出码: $rc ----"
done

FAIL=0
for rc in "${RCS[@]}"; do [ "$rc" -ne 0 ] && FAIL=1; done

echo ""
echo "==================== audit:ui 汇总 ===================="
echo "BASE=$BASE"
for i in "${!NAMES[@]}"; do
  line="${NAMES[$i]}: 退出码=${RCS[$i]} 报告=${OUTS[$i]}"
  echo "$line"
done
if [ "$FAIL" -eq 0 ]; then
  echo "结果: PASS"
else
  echo "结果: FAIL（任一子检查失败即整体失败）"
fi
exit "$FAIL"
