#!/usr/bin/env bash
# HotSpot 可访问性门禁（axe-core 逐页扫描）。
#
# 前置：dev 环境已启动（默认 http://localhost:8888 或 worktree 网关）；已安装 agent-browser；
#       frontend 已 bun install（axe-core 为 devDependency，脚本从 node_modules 注入）。
#
# 用法：
#   bun run audit:a11y
#   bun run audit:a11y -- --base http://localhost:8898 --theme forest --mode dark --limit 5
#   bun run audit:a11y -- --mobile --reduced
# 参数：--base <url> --session <name> --theme vibrant|forest|inkwash --mode light|dark
#       --reduced（reduced-motion 检查，可与 --mobile 组合）--mobile（追加 390×844 5 页）
#       --limit <n>（调试：每组只跑前 n 页）--help
# 输出：<仓库根>/.workwork/browser-check/a11y-<时间戳>/（report.json + summary.txt + raw/）
# 退出码：serious/critical > 0、axe 错误、页面加载失败、--reduced 有残留 → 1。

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/qa-common.sh"
find_ab
find_node

BASE="${A11Y_BASE:-http://localhost:8888}"
SESSION="${A11Y_SESSION:-hotspot-audit-a11y-$$}"
THEME="${A11Y_THEME:-vibrant}"
MODE="${A11Y_MODE:-light}"
REDUCED=0
MOBILE=0
LIMIT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
    --theme) THEME="${2:-$THEME}"; shift ;;
    --mode) MODE="${2:-$MODE}"; shift ;;
    --reduced) REDUCED=1 ;;
    --mobile) MOBILE=1 ;;
    --limit) LIMIT="${2:-0}"; shift ;;
    --help|-h) sed -n '2,20p' "$0" | sed 's/^# *//'; exit 0 ;;
    *) echo "未知参数: $1（--help 查看用法）"; exit 2 ;;
  esac
  shift
done

case "$THEME" in vibrant|forest|inkwash) ;; *) echo "--theme 仅支持 vibrant|forest|inkwash"; exit 2 ;; esac
case "$MODE" in light|dark) ;; *) echo "--mode 仅支持 light|dark"; exit 2 ;; esac

AXE_TOOL="$REPO_ROOT/frontend/node_modules/axe-core/axe.min.js"
if [ ! -f "$AXE_TOOL" ]; then
  echo "找不到 $AXE_TOOL，请先 cd frontend && bun install"; exit 1
fi

OUT="$(make_out_dir a11y)"
: > "$OUT/raw/a11y.jsonl"
: > "$OUT/raw/reduced.jsonl"

ab() { "$AB_BIN" --init-script "$AXE_TOOL" --session "$SESSION" "$@"; }
cleanup() { ab close >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "HotSpot a11y 审计"
echo "BASE=$BASE SESSION=$SESSION THEME=$THEME MODE=$MODE REDUCED=$REDUCED MOBILE=$MOBILE"
echo "输出：$OUT"

ensure_login
ab eval "localStorage.setItem('appTheme','$THEME'); localStorage.setItem('colorScheme','$MODE'); true" >/dev/null 2>&1
if [ "$REDUCED" = "1" ]; then
  ab set media light reduced-motion >/dev/null 2>&1
else
  ab set media light no-preference >/dev/null 2>&1
fi

build_page_lists
DESK=("${DESKTOP_PAGES[@]}")
MOB=()
[ "$MOBILE" = "1" ] && MOB=("${MOBILE_PAGES[@]}")
if [ "$LIMIT" -gt 0 ]; then
  DESK=("${DESK[@]:0:$LIMIT}")
  [ "${#MOB[@]}" -gt 0 ] && MOB=("${MOB[@]:0:$LIMIT}")
fi

TOTAL=$(( ${#DESK[@]} + ${#MOB[@]} ))
idx=0

run_group() { # $1=label $2=vw $3=vh，其余为 route
  local label="$1" vw="$2" vh="$3"; shift 3
  ab set viewport "$vw" "$vh" >/dev/null 2>&1
  for route in "$@"; do
    idx=$((idx + 1))
    echo "[$idx/$TOTAL] $label $route"
    if ! ab open "$BASE$route" >/dev/null 2>&1; then
      echo "  OPEN FAIL"
      printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"open-fail"}}\n' "$idx" "$route" "$vw" "$THEME" "$MODE" >> "$OUT/raw/a11y.jsonl"
      continue
    fi
    if ! ab wait --load networkidle >/dev/null 2>&1; then
      printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"load-fail"}}\n' "$idx" "$route" "$vw" "$THEME" "$MODE" >> "$OUT/raw/a11y.jsonl"
      continue
    fi
    ab wait 1700 >/dev/null 2>&1
    local axe_ready
    axe_ready="$(ab eval 'typeof axe' 2>/dev/null | tail -1 | tr -d '"')"
    # 注入脚本偶发未生效（/map 首屏资源重、并发时曾出现 axe-not-loaded）：重载一次重试，
    # 仍失败才按错误记录（不静默跳过）。
    if [ "$axe_ready" != "function" ] && [ "$axe_ready" != "object" ]; then
      ab open "$BASE$route" >/dev/null 2>&1
      ab wait --load networkidle >/dev/null 2>&1
      ab wait 1200 >/dev/null 2>&1
      axe_ready="$(ab eval 'typeof axe' 2>/dev/null | tail -1 | tr -d '"')"
    fi
    if [ "$axe_ready" != "function" ] && [ "$axe_ready" != "object" ]; then
      printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"axe-not-loaded"}}\n' "$idx" "$route" "$vw" "$THEME" "$MODE" >> "$OUT/raw/a11y.jsonl"
      continue
    fi
    cat "$SCRIPT_DIR/lib/a11y-run.js" | ab eval --stdin >/dev/null 2>&1
    local axe_raw="null" try_count=0
    while [ "$try_count" -lt 25 ]; do
      axe_raw="$(ab eval 'window.__axe' 2>/dev/null | tail -1)"
      if [ -n "$axe_raw" ] && [ "$axe_raw" != "null" ] && [ "$axe_raw" != '""' ]; then break; fi
      sleep 0.3
      try_count=$((try_count + 1))
    done
    if [ -z "$axe_raw" ] || [ "$axe_raw" = "null" ]; then
      axe_raw='{"error":"axe-timeout"}'
    fi
    printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":%s}\n' "$idx" "$route" "$vw" "$THEME" "$MODE" "$axe_raw" >> "$OUT/raw/a11y.jsonl"
    if [ "$REDUCED" = "1" ]; then
      local reduced_raw
      reduced_raw="$(cat "$SCRIPT_DIR/lib/reduced-check.js" | ab eval --stdin 2>/dev/null | tail -1)"
      printf '{"slug":"%s","route":"%s","reduced":%s}\n' "$idx" "$route" "$reduced_raw" >> "$OUT/raw/reduced.jsonl"
    fi
  done
}

[ "${#DESK[@]}" -gt 0 ] && run_group desktop 1440 900 "${DESK[@]}"
[ "${#MOB[@]}" -gt 0 ] && run_group mobile 390 844 "${MOB[@]}"

"$RUNNER" "$SCRIPT_DIR/lib/a11y-summary.mjs" "$OUT" "$REDUCED"
RC=$?
echo "完成：$OUT"
exit "$RC"
