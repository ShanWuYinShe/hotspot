#!/usr/bin/env bash
# HotSpot 触控热区门禁（390×844 逐页扫描，<44 离群对照 allowlist）。
#
# 前置：dev 环境已启动；已安装 agent-browser。
#
# 用法：
#   bun run audit:touch
#   bun run audit:touch -- --base http://localhost:8898 --limit 5
#   bun run audit:touch -- --allowlist /tmp/allowlist.json --limit 5   # 负向验证/临时口径
#   bun run audit:touch -- --baseline                                  # 仅 r5 基线 22 页（快速/对照）
# 参数：--base <url> --session <name> --allowlist <path> --limit <n> --baseline --help
# 默认：全量页面清单（QA 18 + 审计扩展 17 + 详情页，桌面路由在 390 下扫描）。
# 输出：<仓库根>/.workwork/browser-check/touch-<时间戳>/（report.json + summary.txt + raw/）
# 退出码：存在未命中 allowlist 的离群、页面加载失败 → 1。
# allowlist：scripts/ui-quality-touch-allowlist.json（稳定 selector + 理由；维护方式见 docs/frontend-ui-audit.md）。

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/qa-common.sh"
find_ab
find_node

BASE="${TOUCH_BASE:-http://localhost:8888}"
SESSION="${TOUCH_SESSION:-hotspot-audit-touch-$$}"
ALLOWLIST="$SCRIPT_DIR/ui-quality-touch-allowlist.json"
LIMIT=0
FULL=1
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
    --allowlist) ALLOWLIST="${2:-$ALLOWLIST}"; shift ;;
    --limit) LIMIT="${2:-0}"; shift ;;
    --full) FULL=1 ;;
    --baseline) FULL=0 ;;
    --help|-h) sed -n '2,20p' "$0" | sed 's/^# *//'; exit 0 ;;
    *) echo "未知参数: $1（--help 查看用法）"; exit 2 ;;
  esac
  shift
done
[ -f "$ALLOWLIST" ] || { echo "找不到 allowlist: $ALLOWLIST"; exit 1; }

OUT="$(make_out_dir touch)"
: > "$OUT/raw/touch.jsonl"

ab() { "$AB_BIN" --session "$SESSION" "$@"; }
cleanup() { ab close >/dev/null 2>&1 || true; }
trap cleanup EXIT

if [ "$FULL" = "1" ]; then echo "HotSpot 触控热区审计（390x844，全量页面）"; else echo "HotSpot 触控热区审计（390x844，r5 基线页面）"; fi
echo "BASE=$BASE SESSION=$SESSION ALLOWLIST=$ALLOWLIST"
echo "输出：$OUT"

ensure_login
ab set viewport 390 844 >/dev/null 2>&1

build_page_lists
if [ "$FULL" = "1" ]; then
  PAGES=("${DESKTOP_PAGES[@]}")
else
  build_touch_pages
  PAGES=("${TOUCH_PAGES[@]}")
fi
if [ "$LIMIT" -gt 0 ]; then
  PAGES=("${PAGES[@]:0:$LIMIT}")
fi

ALLOW_JSON="$(cat "$ALLOWLIST")"
idx=0
for route in "${PAGES[@]}"; do
  idx=$((idx + 1))
  echo "[$idx/${#PAGES[@]}] $route"
  if ! ab open "$BASE$route" >/dev/null 2>&1; then
    echo "  OPEN FAIL"
    printf '{"slug":"%s","route":"%s","elements":{"error":"open-fail"}}\n' "$idx" "$route" >> "$OUT/raw/touch.jsonl"
    continue
  fi
  if ! ab wait --load networkidle >/dev/null 2>&1; then
    printf '{"slug":"%s","route":"%s","elements":{"error":"load-fail"}}\n' "$idx" "$route" >> "$OUT/raw/touch.jsonl"
    continue
  fi
  ab wait 1400 >/dev/null 2>&1
  printf 'window.__TOUCH_ALLOWLIST=%s; "ok"' "$ALLOW_JSON" | ab eval --stdin >/dev/null 2>&1
  elements_raw="$(cat "$SCRIPT_DIR/lib/touch-scan.js" | ab eval --stdin 2>/dev/null | tail -1)"
  if [ -z "$elements_raw" ] || [ "$elements_raw" = "null" ]; then
    elements_raw='{"error":"scan-fail"}'
  fi
  printf '{"slug":"%s","route":"%s","elements":%s}\n' "$idx" "$route" "$elements_raw" >> "$OUT/raw/touch.jsonl"
done

"$RUNNER" "$SCRIPT_DIR/lib/touch-summary.mjs" "$OUT" "$ALLOWLIST"
RC=$?
echo "完成：$OUT"
exit "$RC"
