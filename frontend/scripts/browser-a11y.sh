#!/usr/bin/env bash
# HotSpot 可访问性门禁（axe-core 逐页扫描）。
#
# 前置：dev 环境已启动（默认 http://localhost:8888 或 worktree 网关）；已安装 agent-browser；
#       frontend 已 bun install（axe-core 为 devDependency，脚本从 node_modules 注入）。
#
# 用法：
#   bun run audit:a11y
#   bun run audit:a11y -- --base http://localhost:8899 --theme forest --mode dark --limit 5
#   bun run audit:a11y -- --mobile --reduced
#   bun run audit:a11y -- --matrix                     # 3 主题 × 2 明暗 × 桌面全量 + 移动抽测
#   bun run audit:a11y -- --matrix --only /admin,/notify  # 矩阵 + 路由子串过滤（调试/专项）
# 参数：--base <url> --session <name> --theme vibrant|forest|inkwash --mode light|dark
#       --reduced（reduced-motion 检查，可与 --mobile 组合）--mobile（追加 390×844 5 页）
#       --matrix（主题×明暗矩阵；矩阵下忽略 --theme/--mode/--mobile）
#       --only <csv>（调试：只跑路由含任一子串的页面）--limit <n>（每组前 n 页）
#       --parallel <n>（矩阵并行 worker 数，默认 1；2 可显著省时，各 worker 独立 session）--help
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
MATRIX=0
LIMIT=0
ONLY=""
PARALLEL=1
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
    --theme) THEME="${2:-$THEME}"; shift ;;
    --mode) MODE="${2:-$MODE}"; shift ;;
    --reduced) REDUCED=1 ;;
    --mobile) MOBILE=1 ;;
    --matrix) MATRIX=1 ;;
    --only) ONLY="${2:-}"; shift ;;
    --limit) LIMIT="${2:-0}"; shift ;;
    --parallel) PARALLEL="${2:-1}"; shift ;;
    --help|-h) sed -n '2,24p' "$0" | sed 's/^# *//'; exit 0 ;;
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
echo "BASE=$BASE SESSION=$SESSION THEME=$THEME MODE=$MODE REDUCED=$REDUCED MOBILE=$MOBILE MATRIX=$MATRIX"
[ -n "$ONLY" ] && echo "ONLY=$ONLY"
[ "$PARALLEL" -gt 1 ] && echo "PARALLEL=$PARALLEL"
echo "输出：$OUT"

ensure_login
if [ "$REDUCED" = "1" ] && [ "$MATRIX" != "1" ]; then
  ab set media light reduced-motion >/dev/null 2>&1
else
  ab set media light no-preference >/dev/null 2>&1
fi

build_page_lists
DESK=("${DESKTOP_PAGES[@]}")
# 哨兵：bash 3.2 + set -u 下空数组展开会报 unbound；单组合无移动页时必须是非空数组。
MOB=("__none__")
if [ "$MATRIX" = "1" ] || [ "$MOBILE" = "1" ]; then
  MOB=("${MOBILE_PAGES[@]}")
fi

if [ -n "$ONLY" ]; then
  IFS=',' read -r -a ONLY_PARTS <<< "$ONLY"
  FILTERED_DESK=()
  DESK_MATCH=0
  for r in "${DESK[@]}"; do
    for part in "${ONLY_PARTS[@]}"; do
      [ -z "$part" ] && continue
      case "$r" in *"$part"*) FILTERED_DESK+=("$r"); DESK_MATCH=$((DESK_MATCH + 1)); break ;; esac
    done
  done
  if [ "$DESK_MATCH" -gt 0 ]; then DESK=("${FILTERED_DESK[@]}"); else DESK=("__none__"); fi
  MOB_MATCH=0
  FILTERED_MOB=()
  for r in "${MOB[@]}"; do
    for part in "${ONLY_PARTS[@]}"; do
      [ -z "$part" ] && continue
      case "$r" in *"$part"*) FILTERED_MOB+=("$r"); MOB_MATCH=$((MOB_MATCH + 1)); break ;; esac
    done
  done
  if [ "$MOB_MATCH" -gt 0 ]; then MOB=("${FILTERED_MOB[@]}"); else MOB=("__none__"); fi
fi

if [ "$LIMIT" -gt 0 ]; then
  DESK=("${DESK[@]:0:$LIMIT}")
  if [ ${#MOB[@]} -gt 0 ]; then MOB=("${MOB[@]:0:$LIMIT}"); fi
fi

THEMES=("$THEME")
MODES=("$MODE")
if [ "$MATRIX" = "1" ]; then
  THEMES=(vibrant forest inkwash)
  MODES=(light dark)
fi

if [ "${DESK[0]}" = "__none__" ]; then DESK_N=0; else DESK_N=${#DESK[@]}; fi
if [ "${MOB[0]}" = "__none__" ]; then MOB_N=0; else MOB_N=${#MOB[@]}; fi
if [ "$DESK_N" = "0" ] && [ "$MOB_N" = "0" ]; then
  echo "没有匹配的页面（检查 --only/--limit）"; exit 2
fi
TOTAL=$(( ${#THEMES[@]} * ${#MODES[@]} * (DESK_N + MOB_N) ))
idx=0
CUR_THEME="$THEME"
CUR_MODE="$MODE"

run_page() { # $1=route $2=vw $3=vh
  local route="$1" vw="$2" vh="$3"
  idx=$((idx + 1))
  echo "[$idx/$TOTAL] $CUR_THEME/$CUR_MODE ${vw}x${vh} $route"
  if ! ab open "$BASE$route" >/dev/null 2>&1; then
    echo "  OPEN FAIL"
    printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"open-fail"}}\n' "$idx" "$route" "$vw" "$CUR_THEME" "$CUR_MODE" >> "$RAW_FILE"
    return
  fi
  # 矩阵/并行下 networkidle 可能因 WebSocket/并发永不触发（实测卡 /messages）：矩阵用固定等待，
  # 单组合保留 networkidle。页面 open 已等 load，短暂等待足够 axe 在 DOM 稳定后执行。
  if [ "$MATRIX" = "1" ]; then
    ab wait 1400 >/dev/null 2>&1
  elif ! ab wait --load networkidle >/dev/null 2>&1; then
    printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"load-fail"}}\n' "$idx" "$route" "$vw" "$CUR_THEME" "$CUR_MODE" >> "$RAW_FILE"
    return
  fi
  # 矩阵模式页面多，压缩固定等待（axe 在 DOM 稳定后跑）；单组合保留更长等待。
  if [ "$MATRIX" = "1" ]; then ab wait 900 >/dev/null 2>&1; else ab wait 1700 >/dev/null 2>&1; fi
  # reduced-motion 必须每页重设：agent-browser 的 `open` 打开新标签页会重置 CDP 媒体模拟，
  # 只在循环外 set 一次时，reduced 检查实际在 no-preference 下执行——会把 ambient 装饰动画
  # 误报成长动画（R14 实测 37 页假阳性）。放在页面稳定后、注入检查脚本前设置。
  if [ "$REDUCED" = "1" ] && [ "$MATRIX" != "1" ]; then
    ab set media light reduced-motion >/dev/null 2>&1
  fi
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
    printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":{"error":"axe-not-loaded"}}\n' "$idx" "$route" "$vw" "$CUR_THEME" "$CUR_MODE" >> "$RAW_FILE"
    return
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
  printf '{"slug":"%s","route":"%s","vw":%s,"theme":"%s","mode":"%s","axe":%s}\n' "$idx" "$route" "$vw" "$CUR_THEME" "$CUR_MODE" "$axe_raw" >> "$RAW_FILE"
  if [ "$REDUCED" = "1" ] && [ "$MATRIX" != "1" ]; then
    local reduced_raw
    reduced_raw="$(cat "$SCRIPT_DIR/lib/reduced-check.js" | ab eval --stdin 2>/dev/null | tail -1)"
    printf '{"slug":"%s","route":"%s","reduced":%s}\n' "$idx" "$route" "$reduced_raw" >> "$REDUCED_RAW_FILE"
  fi
}

run_group() { # $1=label $2=vw $3=vh，其余 route
  local label="$1" vw="$2" vh="$3"; shift 3
  ab set viewport "$vw" "$vh" >/dev/null 2>&1
  local route
  for route in "$@"; do
    [ -n "$route" ] || continue
    [ "$route" = "__none__" ] && continue
    run_page "$route" "$vw" "$vh"
  done
}

# 单组合 / 矩阵统一走 run_combos；矩阵可用 --parallel N 分 worker 并行（各自独立 session + raw 文件）。
RAW_FILE="$OUT/raw/a11y.jsonl"
REDUCED_RAW_FILE="$OUT/raw/reduced.jsonl"
: > "$RAW_FILE"
: > "$REDUCED_RAW_FILE"

COMBOS=()
for t in "${THEMES[@]}"; do
  for m in "${MODES[@]}"; do COMBOS+=("$t|$m"); done
done

run_combos() { # 参数：组合（theme|mode）
  local combo t m
  for combo in "$@"; do
    t="${combo%%|*}"
    m="${combo##*|}"
    CUR_THEME="$t"; CUR_MODE="$m"
    echo "=== 组合 $t/$m（桌面 $DESK_N 页 + 移动 $MOB_N 页，worker=$WORKER_ID）==="
    ab eval "localStorage.setItem('appTheme','$t'); localStorage.setItem('colorScheme','$m'); true" >/dev/null 2>&1
    ab open "$BASE/" >/dev/null 2>&1
    # 组合切换：矩阵/并行下 networkidle 可能永不触发（WebSocket），用固定等待
    if [ "$MATRIX" = "1" ]; then ab wait 1400 >/dev/null 2>&1; else ab wait --load networkidle >/dev/null 2>&1; fi
    [ "$DESK_N" -gt 0 ] && run_group desktop 1440 900 "${DESK[@]}"
    [ "$MOB_N" -gt 0 ] && run_group mobile 390 844 "${MOB[@]}"
  done
}

WORKER_ID=0
if [ "$MATRIX" = "1" ] && [ "$PARALLEL" -gt 1 ]; then
  echo "矩阵并行：$PARALLEL worker（各 worker 独立 session/输出文件）"
  WORKER_LOGS=()
  WORKER_PIDS=()
  for ((w = 0; w < PARALLEL; w++)); do
    (
      export SESSION="${SESSION}-w$w"
      WORKER_ID=$w
      RAW_FILE="$OUT/raw/a11y-w$w.jsonl"
      REDUCED_RAW_FILE="$OUT/raw/reduced-w$w.jsonl"
      : > "$RAW_FILE"
      : > "$REDUCED_RAW_FILE"
      trap 'ab close >/dev/null 2>&1 || true' EXIT
      my_combos=()
      for ((ci = w; ci < ${#COMBOS[@]}; ci += PARALLEL)); do my_combos+=("${COMBOS[$ci]}"); done
      run_combos "${my_combos[@]}"
    ) > "$OUT/raw/worker-$w.log" 2>&1 &
    WORKER_PIDS+=($!)
    WORKER_LOGS+=("$OUT/raw/worker-$w.log")
  done
  FAILED=0
  for pid in "${WORKER_PIDS[@]}"; do wait "$pid" || FAILED=1; done
  if [ "$FAILED" = "1" ]; then
    echo "部分 worker 执行异常，日志："
    for f in "${WORKER_LOGS[@]}"; do tail -5 "$f"; done
  fi
else
  run_combos "${COMBOS[@]}"
fi

"$RUNNER" "$SCRIPT_DIR/lib/a11y-summary.mjs" "$OUT" "$REDUCED"
RC=$?
echo "完成：$OUT"
exit "$RC"