#!/bin/sh
# 性能预算回归门禁：对运行中的目标实例跑固定读端点，与 docs/perf-baseline.json
# 比较 rps 下限与 p99 上限；任一回归即非零退出。用于「改完性能敏感代码后
# 本机回归」或发版前手动验收（不进默认 CI，避免共享 runner 噪声）。
#
# 用法：
#   ./scripts/perf-budget.sh                       # 默认 http://localhost:8082（backend-test）
#   ./scripts/perf-budget.sh --base http://localhost:8888 --duration 10s
#   ./scripts/perf-budget.sh --update-baseline     # 有意提升/调整基线时刷新 JSON
#   ./scripts/perf-budget.sh --max-p99-mult 1.5 --min-rps-ratio 0.8
#
# 口径：rps 必须 >= baseline.rps * min_rps_ratio；p99 必须 <= baseline.p99 * max_p99_multiplier。
# 业务码/传输错误由 perf-smoke.sh 自身校验（此处直接透传其退出码）。
set -eu

BASE="${BASE:-http://localhost:8082}"
BASELINE="${BASELINE:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)/docs/perf-baseline.json}"
CONCURRENCY="${CONCURRENCY:-10}"
DURATION="${DURATION:-5s}"
MIN_RPS_RATIO="${MIN_RPS_RATIO:-}"
MAX_P99_MULT="${MAX_P99_MULT:-}"
UPDATE_BASELINE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="$2"; shift 2 ;;
    --baseline) BASELINE="$2"; shift 2 ;;
    --concurrency) CONCURRENCY="$2"; shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    --min-rps-ratio) MIN_RPS_RATIO="$2"; shift 2 ;;
    --max-p99-mult) MAX_P99_MULT="$2"; shift 2 ;;
    --update-baseline) UPDATE_BASELINE=1; shift ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
REPORT="$(mktemp)"
trap 'rm -f "$REPORT"' EXIT

echo "==> perf-smoke（base=$BASE c=$CONCURRENCY d=${DURATION}）"
"$ROOT_DIR/scripts/perf-smoke.sh" --base "$BASE" --concurrency "$CONCURRENCY" --duration "$DURATION" --report "$REPORT"

python3 - "$BASELINE" "$REPORT" "$MIN_RPS_RATIO" "$MAX_P99_MULT" "$UPDATE_BASELINE" "$BASE" "$CONCURRENCY" "$DURATION" <<'PY'
import json
import os
import sys
from datetime import date

baseline_path, report_path, ratio_arg, mult_arg, update_arg, base, conc, dur = sys.argv[1:9]
rows = {}
with open(report_path, encoding='utf-8') as f:
    for line in f:
        parts = line.strip().split(',')
        if len(parts) != 5:
            continue
        name, rps, p50, p95, p99 = parts
        try:
            rows[name] = {'rps': float(rps), 'p50_ms': float(p50), 'p95_ms': float(p95), 'p99_ms': float(p99)}
        except ValueError:
            continue

if update_arg == '1':
    data = {
        'environment': 'backend-test / hotspot_test（含 E2E 累计数据；PGroonga + nearby 物化后的基线）',
        'host_note': 'Apple Silicon 本机 Docker；8 vCPU；postgres 容器 2 CPU 限额',
        'workload': f'perf-smoke：c={conc}，warmup=2s，duration={dur}',
        'updated_at': date.today().isoformat(),
        'tolerance': {'min_rps_ratio': 0.6, 'max_p99_multiplier': 2.5},
        'endpoints': {name: {'rps': round(v['rps'], 1), 'p99_ms': round(v['p99_ms'], 2)} for name, v in sorted(rows.items())},
    }
    with open(baseline_path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write('\n')
    print(f'==> 基线已更新：{baseline_path}（{len(rows)} 端点）')
    sys.exit(0)

with open(baseline_path, encoding='utf-8') as f:
    baseline = json.load(f)
eps = baseline.get('endpoints', {})
min_ratio = float(ratio_arg) if ratio_arg else float(baseline.get('tolerance', {}).get('min_rps_ratio', 0.6))
max_mult = float(mult_arg) if mult_arg else float(baseline.get('tolerance', {}).get('max_p99_multiplier', 2.5))

print('')
print('%-26s %12s %12s %12s %12s %s' % ('endpoint', 'rps', 'rps_floor', 'p99_ms', 'p99_ceiling', 'status'))
failures = []
missing = []
for name, exp in eps.items():
    cur = rows.get(name)
    if cur is None:
        missing.append(name)
        continue
    rps_floor = exp['rps'] * min_ratio
    p99_ceiling = exp['p99_ms'] * max_mult
    ok = cur['rps'] >= rps_floor and cur['p99_ms'] <= p99_ceiling
    status = 'OK' if ok else 'REGRESSION'
    if not ok:
        failures.append((name, cur, rps_floor, p99_ceiling))
    print('%-26s %12.1f %12.1f %12.2f %12.2f %s' % (name, cur['rps'], rps_floor, cur['p99_ms'], p99_ceiling, status))

if missing:
    print('')
    print('!! 缺失基线端点（本次 perf-smoke 未产出）：' + ', '.join(missing), file=sys.stderr)
if failures or missing:
    print('')
    print(f'PERF BUDGET FAILED：{len(failures)} 项回归、{len(missing)} 项缺失', file=sys.stderr)
    sys.exit(1)
print('')
print(f'PERF BUDGET OK：{len(eps)} 个端点均在预算内（rps>={min_ratio:.0%} baseline，p99<={max_mult:g}x baseline）')
PY
