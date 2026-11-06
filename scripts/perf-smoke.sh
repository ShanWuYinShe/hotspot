#!/bin/sh
# 后端只读性能冒烟/基线：对目标实例跑一组固定读端点（访客身份），输出
# rps/p50/p95/p99 表格，并在业务码异常或超过阈值时以非零退出码失败。
#
# 用法：
#   ./scripts/perf-smoke.sh                         # 默认 http://localhost:8082（backend-test）
#   ./scripts/perf-smoke.sh --base http://localhost:8888 --concurrency 20 --duration 10s
#   TOKEN=<access_token> ./scripts/perf-smoke.sh    # 复用既有 token（非测试环境必须显式传）
#   ./scripts/perf-smoke.sh --min-rps 100 --max-p99-ms 200   # CI 门禁模式
#
# 环境变量/参数：BASE / TOKEN / CONCURRENCY / DURATION / WARMUP / MIN_RPS /
# MAX_P99_MS / REF_LAT / REF_LNG / SEARCH_KEYWORD / MOMENTS_SEARCH_KEYWORD /
# GROUPS_SEARCH_KEYWORD / LOADGEN_BIN。
# 默认附近坐标与搜索关键词对齐 dev 种子数据（徐州云龙湖 / 用户昵称「林一」），
# 避免历史默认「旧金山 + abc」导致附近/搜索测到空结果集。
#
# 判定口径：读端点成功响应必须全部为 HTTP 200 且 envelope code=0；出现业务
# 错误码、传输错误或低于 MIN_RPS/高于 MAX_P99_MS（阈值默认关闭）即退出 1。
set -eu

BASE="${BASE:-http://localhost:8082}"
CONCURRENCY="${CONCURRENCY:-10}"
DURATION="${DURATION:-10s}"
WARMUP="${WARMUP:-2s}"
MIN_RPS="${MIN_RPS:-0}"
MAX_P99_MS="${MAX_P99_MS:-0}"
# 种子锚点：backend-go/internal/seed/specs.go seedAnchorLatitude/Longitude
REF_LAT="${REF_LAT:-34.2604}"
REF_LNG="${REF_LNG:-117.1855}"
# 种子用户「林一」昵称命中 users/search（旧默认 abc 与种子数据不匹配，空结果集无效）
SEARCH_KEYWORD="${SEARCH_KEYWORD:-林}"
MOMENTS_SEARCH_KEYWORD="${MOMENTS_SEARCH_KEYWORD:-游客}"
GROUPS_SEARCH_KEYWORD="${GROUPS_SEARCH_KEYWORD:-群}"

while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="$2"; shift 2 ;;
    --concurrency) CONCURRENCY="$2"; shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    --warmup) WARMUP="$2"; shift 2 ;;
    --no-warmup) WARMUP="0" ;;
    --min-rps) MIN_RPS="$2"; shift 2 ;;
    --max-p99-ms) MAX_P99_MS="$2"; shift 2 ;;
    --keyword) SEARCH_KEYWORD="$2"; MOMENTS_SEARCH_KEYWORD="$2"; shift 2 ;;
    --lat) REF_LAT="$2"; shift 2 ;;
    --lng) REF_LNG="$2"; shift 2 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
LOADGEN_BIN="${LOADGEN_BIN:-}"
if [ -z "$LOADGEN_BIN" ]; then
  TMP_DIR="$(mktemp -d)"
  LOADGEN_BIN="$TMP_DIR/loadgen"
  ( cd "$ROOT_DIR/backend-go" && go build -o "$LOADGEN_BIN" ./cmd/loadgen )
  trap 'rm -rf "$TMP_DIR"' EXIT
fi

if [ -z "${TOKEN:-}" ]; then
  TOKEN=$(curl -sS -m 10 -X POST -H 'Content-Type: application/json' \
    -H "Idempotency-Key: perf-smoke-$(date +%s)-000000000000000000" \
    -d '{"client_type":"app","device_id":"perfsmoke"}' \
    "$BASE/public/v1/auth/guest" 2>/dev/null \
    | python3 -c 'import json,sys
try:
    print(json.load(sys.stdin).get("data",{}).get("access_token",""))
except Exception:
    pass' 2>/dev/null || true)
  if [ -z "$TOKEN" ]; then
    echo "获取访客 token 失败（$BASE 不可达或未启用 guest 登录）；非测试环境请显式传 TOKEN=<access_token>" >&2
    exit 1
  fi
fi

AUTH="Authorization: Bearer $TOKEN"
FAILURES=0
printf '%-26s %10s %9s %9s %9s %s\n' endpoint rps p50_ms p95_ms p99_ms codes

# 单端点压测：先预热（丢弃结果）再采样；返回码/阈值不满足时记失败。
run() {
  name="$1"; path="$2"; body="$3"
  if [ "$WARMUP" != "0" ]; then
    "$LOADGEN_BIN" -url "$BASE$path" -method POST -body "$body" -H "$AUTH" \
      -c "$CONCURRENCY" -d "$WARMUP" >/dev/null 2>&1 || true
  fi
  raw=$("$LOADGEN_BIN" -url "$BASE$path" -method POST -body "$body" -H "$AUTH" \
    -c "$CONCURRENCY" -d "$DURATION" 2>&1) || {
    echo "!! $name: loadgen 执行失败（目标不可达？）" >&2
    FAILURES=$((FAILURES + 1))
    return 0
  }
  rps=$(printf %s "$raw" | grep -o 'rps=[0-9.]*' | head -1 | cut -d= -f2 || true)
  p50=$(printf %s "$raw" | grep -o 'p50=[0-9.]*' | head -1 | cut -d= -f2 || true)
  p95=$(printf %s "$raw" | grep -o 'p95=[0-9.]*' | head -1 | cut -d= -f2 || true)
  p99=$(printf %s "$raw" | grep -o 'p99=[0-9.]*' | head -1 | cut -d= -f2 || true)
  statuses=$(printf %s "$raw" | grep -o 'statuses=map\[[^]]*\]' | head -1 || true)
  codes=$(printf %s "$raw" | grep -o 'codes=map\[[^]]*\]' | head -1 | cut -d= -f2- || true)
  errors=$(printf %s "$raw" | grep -o 'errors=map\[[^]]*\]' | head -1 | cut -d= -f2- || true)
  printf '%-26s %10s %9s %9s %9s %s\n' "$name" "${rps:-n/a}" "${p50:-n/a}" "${p95:-n/a}" "${p99:-n/a}" "${codes:-n/a}"

  # 业务契约：读端点全部 200 且 code=0；无传输错误。
  if [ -n "${errors:-}" ] && [ "$errors" != "map[]" ]; then
    echo "!! $name: 传输错误 $errors" >&2; FAILURES=$((FAILURES + 1))
  fi
  if ! printf '%s' "$statuses" | grep -Eq '^statuses=map\[200:[0-9]+\]$'; then
    echo "!! $name: 非 200 状态码 $statuses" >&2; FAILURES=$((FAILURES + 1))
  fi
  if ! printf '%s' "$codes" | grep -Eq '^map\[0:[0-9]+\]$'; then
    echo "!! $name: 业务错误码 ${codes:-}（读端点应全部 code=0）" >&2; FAILURES=$((FAILURES + 1))
  fi
  if [ "$MIN_RPS" != "0" ] && ! awk -v v="${rps:-0}" -v m="$MIN_RPS" 'BEGIN{exit !(v+0 >= m+0)}'; then
    echo "!! $name: rps ${rps:-0} < MIN_RPS $MIN_RPS" >&2; FAILURES=$((FAILURES + 1))
  fi
  if [ "$MAX_P99_MS" != "0" ] && ! awk -v v="${p99:-0}" -v m="$MAX_P99_MS" 'BEGIN{exit !(v+0 <= m+0)}'; then
    echo "!! $name: p99 ${p99:-0}ms > MAX_P99_MS $MAX_P99_MS" >&2; FAILURES=$((FAILURES + 1))
  fi
}

run users/account /api/v1/users/account '{}'
run system/bootstrap /api/v1/system/bootstrap '{}'
run notify/fetch /api/v1/notify/fetch '{"pagination":{"limit":20}}'
run moments/fetch_following /api/v1/moments/fetch_following '{"pagination":{"limit":20}}'
NEARBY_BODY=$(printf '{"reference_point":{"latitude":%s,"longitude":%s},"radius":5000,"pagination":{"limit":20}}' "$REF_LAT" "$REF_LNG")
run moments/fetch_nearby /api/v1/moments/fetch_nearby "$NEARBY_BODY"
SEARCH_BODY=$(printf '{"keyword":"%s","pagination":{"limit":20}}' "$SEARCH_KEYWORD")
run users/search /api/v1/users/search "$SEARCH_BODY"
MOMENT_SEARCH_BODY=$(printf '{"keyword":"%s","sort_by":"newest","pagination":{"limit":20}}' "$MOMENTS_SEARCH_KEYWORD")
run moments/search /api/v1/moments/search "$MOMENT_SEARCH_BODY"
GROUP_SEARCH_BODY=$(printf '{"keyword":"%s","pagination":{"limit":20}}' "$GROUPS_SEARCH_KEYWORD")
run groups/search /api/v1/groups/search "$GROUP_SEARCH_BODY"

if [ "$FAILURES" -ne 0 ]; then
  echo "" >&2
  echo "PERF SMOKE FAILED：$FAILURES 项不满足（rps/p99 阈值或业务码异常）" >&2
  exit 1
fi
echo ""
echo "PERF SMOKE OK：8 个读端点全部返回 code=0（warmup=${WARMUP} duration=${DURATION} concurrency=${CONCURRENCY}）"
