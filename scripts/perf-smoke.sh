#!/bin/sh
# 后端只读性能冒烟：对指定实例跑一组固定读端点（访客身份），输出 rps/p50/p95/p99 表格。
#
# 用法：
#   ./scripts/perf-smoke.sh                        # 默认 http://localhost:8082（backend-test）
#   ./scripts/perf-smoke.sh --base http://localhost:8888 --concurrency 20 --duration 10s
#   TOKEN=<access_token> ./scripts/perf-smoke.sh   # 复用既有 token（非测试环境必须显式传）
#   ./scripts/perf-smoke.sh > /tmp/perf.txt        # 需要存档时自行重定向
#
# 环境变量：BASE / TOKEN / CONCURRENCY / DURATION / LOADGEN_BIN / REF_LAT / REF_LNG / SEARCH_KEYWORD
# 附近流坐标默认取测试种子坐标（37.7749,-122.4194），可用 REF_LAT/REF_LNG 覆盖。
set -eu

BASE="${BASE:-http://localhost:8082}"
CONCURRENCY="${CONCURRENCY:-10}"
DURATION="${DURATION:-10s}"
REF_LAT="${REF_LAT:-37.7749}"
REF_LNG="${REF_LNG:--122.4194}"
SEARCH_KEYWORD="${SEARCH_KEYWORD:-abc}"

while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="$2"; shift 2 ;;
    --concurrency) CONCURRENCY="$2"; shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
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
    "$BASE/public/v1/auth/guest" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("data",{}).get("access_token",""))')
  if [ -z "$TOKEN" ]; then
    echo '获取访客 token 失败；非测试环境请显式传 TOKEN=<access_token>' >&2
    exit 1
  fi
fi

AUTH="Authorization: Bearer $TOKEN"
printf '%-24s %10s %10s %10s %10s %s\n' endpoint rps p50_ms p95_ms p99_ms codes

run() {
  name="$1"; path="$2"; body="$3"
  raw=$("$LOADGEN_BIN" -url "$BASE$path" -method POST -body "$body" -H "$AUTH" -c "$CONCURRENCY" -d "$DURATION" 2>&1)
  rps=$(printf %s "$raw" | grep -o 'rps=[0-9]*' | head -1 | cut -d= -f2)
  p50=$(printf %s "$raw" | grep -o 'p50=[0-9.]*' | head -1 | cut -d= -f2)
  p95=$(printf %s "$raw" | grep -o 'p95=[0-9.]*' | head -1 | cut -d= -f2)
  p99=$(printf %s "$raw" | grep -o 'p99=[0-9.]*' | head -1 | cut -d= -f2)
  codes=$(printf %s "$raw" | grep -o 'codes=map\[[^]]*\]' | head -1)
  printf '%-24s %10s %10s %10s %10s %s\n' "$name" "${rps:-n/a}" "${p50:-n/a}" "${p95:-n/a}" "${p99:-n/a}" "${codes:-n/a}"
}

run users/account /api/v1/users/account '{}'
run system/bootstrap /api/v1/system/bootstrap '{}'
run notify/fetch /api/v1/notify/fetch '{"pagination":{"limit":20}}'
run moments/fetch_following /api/v1/moments/fetch_following '{"pagination":{"limit":20}}'
NEARBY_BODY=$(printf '{"reference_point":{"latitude":%s,"longitude":%s},"radius":5000,"pagination":{"limit":20}}' "$REF_LAT" "$REF_LNG")
run moments/fetch_nearby /api/v1/moments/fetch_nearby "$NEARBY_BODY"
SEARCH_BODY=$(printf '{"keyword":"%s","pagination":{"limit":20}}' "$SEARCH_KEYWORD")
run users/search /api/v1/users/search "$SEARCH_BODY"