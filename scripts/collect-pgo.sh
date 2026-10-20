#!/usr/bin/env bash
# 采集 Go 生产 CPU profile 并生成 PGO 文件（default.pgo）。
#
# 为什么需要：Go 1.21+ 支持 Profile-Guided Optimization —— 编译器用真实热点做内联/分支优化，
# 官方口径 2–14% 提升、零代码改动（https://go.dev/doc/pgo）。
# 关键纪律：**必须用生产/准生产流量采集**。dev 环境的热点分布与线上不同，拿 dev profile
# 编译可能反而变慢 —— 本仓库不提交 dev 采集的 default.pgo。
#
# 用法：
#   ./scripts/collect-pgo.sh                    # 默认 localhost:6060，采集 60s
#   PPROF_ADDR=127.0.0.1:16060 ./scripts/collect-pgo.sh
#   SECONDS_PROFILE=120 ./scripts/collect-pgo.sh
#   采集完 → git add backend-go/cmd/server/default.pgo → 正常 go build 即自动生效
#
# 生效验证：
#   go build ./cmd/server            # Go 自动发现同目录 default.pgo
#   go build -pgo=off ./cmd/server   # 对照组（关 PGO）
#   对比线上 P95 延迟 / QPS，或用 go test -bench + benchstat 对比
#
# 轮换：每次大版本发布前可重采一次；profile 过期（>1-2 个月）收益会衰减。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PPROF_ADDR="${PPROF_ADDR:-127.0.0.1:6060}"
SECONDS_PROFILE="${SECONDS_PROFILE:-60}"
OUT="${REPO_ROOT}/backend-go/cmd/server/default.pgo"

if ! command -v go >/dev/null 2>&1; then
  echo "错误：未找到 go 命令（需要 Go 1.21+）" >&2
  exit 1
fi

echo "==> 检查 pprof 端点 ${PPROF_ADDR}"
if ! curl -sf -o /dev/null --max-time 5 "http://${PPROF_ADDR}/debug/pprof/" 2>/dev/null; then
  echo "错误：pprof 端点不可达。确认后端以 PPROF_ADDR 启动（localhost 绑定，需在宿主机/容器内访问）。" >&2
  echo "提示：容器内运行时可用 'docker exec' 进容器执行，或临时映射端口。" >&2
  exit 1
fi

echo "==> 采集 ${SECONDS_PROFILE}s CPU profile（请确保此刻有真实流量）"
go tool pprof -proto -output="${OUT}" "http://${PPROF_ADDR}/debug/pprof/profile?seconds=${SECONDS_PROFILE}"

if [ ! -s "${OUT}" ]; then
  echo "错误：生成的 profile 为空。" >&2
  exit 1
fi

size=$(wc -c < "${OUT}" | tr -d ' ')
echo "==> 已生成 ${OUT}（${size} bytes）"
echo "==> 下一步：git add backend-go/cmd/server/default.pgo 并正常构建；构建日志里应出现 'using default.pgo'"
