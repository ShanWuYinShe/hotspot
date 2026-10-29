#!/usr/bin/env bash
# ── 开发种子数据（dev 环境）────────────────────────────────────────────
# 在开发 compose 的 backend 容器内执行 cmd/seed：向 Postgres + SeaweedFS 写入一套
# 可复现的测试数据（账号/瞬间/群组/私聊/通知/分享/举报/管理端数据 + 媒体素材）。
#
# 用法（本仓库根目录执行；worktree 内同样适用，compose 项目名按目录名自动隔离）：
#   ./scripts/seed-dev.sh            # 默认 --if-missing：缺失则写入，已存在则跳过
#   ./scripts/seed-dev.sh --reset    # 先按固定前缀清空 seed 数据（含 S3 对象）再重建
#   ./scripts/seed-dev.sh --dry-run  # 只打印写入计划，不写任何数据
#
# 前置：dev compose 已启动（docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d），
# backend 容器 healthy（容器内挂载源码、有 Go SDK 与全部 env，无需额外启动服务）。
# 全新 SeaweedFS 卷缺少对象存储 bucket 时脚本会尝试自动创建（见 ensure_bucket）。
#
# 账号（口令可由 .env / 当前 shell 的 SEED_QA_PASSWORD、SEED_ADMIN_PASSWORD 覆盖）：
#   qa@hotspot.local    Qa12345678      普通用户（QA 巡检主账号，user_seed_lumos）
#   admin@hotspot.local Admin12345678   管理员（grant_all，同时是完整普通用户）
#
# 数据规格/实现：backend-go/internal/seed/；开发素材是外置配套设施（仓库根 devdata/seed/media/，
# 不再 //go:embed 打包进二进制），dev compose 把 devdata/ 挂载到容器 /devdata，脚本显式传
# -assets-dir /devdata/seed/media。素材与数据设施说明见 devdata/seed/README.md。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_ARGS=(-f docker-compose.yml -f docker-compose.dev.yml)

# seed 素材（仓库根 devdata/seed/media）在 dev compose 中挂载到容器 /devdata
# （见 docker-compose.dev.yml 的 backend volumes）。脚本显式传 -assets-dir，
# 不依赖 cmd/seed 的默认候选解析；其它挂载布局可用该 env 覆盖容器内路径。
ASSETS_DIR_CONTAINER="${SEED_ASSETS_DIR_CONTAINER:-/devdata/seed/media}"

# 读取仓库根 .env 中的 SEED_* 口令覆盖值。
# 不能直接 source .env：其中可能含 SMTP_FROM 等带空格/尖括号的值，bash source
# 会解析失败；这里只按 KEY=VALUE 提取需要的两个变量（其余变量由 compose 自己插值）。
env_value() {
	local name="$1"
	[ -f "$ROOT_DIR/.env" ] || return 0
	grep -E "^[[:space:]]*$name=" "$ROOT_DIR/.env" 2>/dev/null |
		tail -n 1 |
		cut -d= -f2- |
		sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//" || true
}

SEED_QA_PASSWORD="${SEED_QA_PASSWORD:-$(env_value SEED_QA_PASSWORD)}"
SEED_ADMIN_PASSWORD="${SEED_ADMIN_PASSWORD:-$(env_value SEED_ADMIN_PASSWORD)}"

# print_header 输出脚本头部注释（帮助文本），不依赖固定的行号范围。
print_header() {
	awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"
}

case "${1:-}" in
	-h | --help)
		print_header
		echo
		echo "实际执行等价于："
		echo '  docker compose -f docker-compose.yml -f docker-compose.dev.yml exec -T backend \'
		echo '    go run ./cmd/seed -assets-dir /devdata/seed/media "$@"'
		exit 0
		;;
esac

if ! command -v docker >/dev/null 2>&1; then
	echo "错误：未找到 docker 命令，无法执行 dev compose。" >&2
	exit 1
fi

# 前置检查：backend 容器必须已在运行（否则 exec 会失败并给出晦涩错误）。
if ! docker compose "${COMPOSE_ARGS[@]}" exec -T backend true </dev/null >/dev/null 2>&1; then
	cat >&2 <<'EOF'
错误：dev compose 的 backend 容器未运行（或 compose 配置无法解析）。

请先在仓库根启动开发环境并等待 backend healthy：
  docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

查看状态：
  docker compose -f docker-compose.yml -f docker-compose.dev.yml ps
EOF
	exit 1
fi

# 首次在全新 SeaweedFS 卷上运行时对象存储 bucket 可能不存在（SeaweedFS S3 网关
# 不会在建桶前接受 PUT）。真实写入前尽力确保 bucket 存在：bucket 名取 backend
# 容器 env S3_BUCKET；自动创建失败不阻塞（外部 S3/预置环境由运维保证，随后
# 的 seed 会给出真实错误）。--dry-run 不写对象存储，跳过此步。
ensure_bucket() {
	local bucket
	bucket="$(docker compose "${COMPOSE_ARGS[@]}" exec -T backend printenv S3_BUCKET 2>/dev/null | tr -d "\r" || true)"
	bucket="${bucket:-hotspot}"
	if echo "s3.bucket.list" | docker compose "${COMPOSE_ARGS[@]}" exec -T seaweedfs weed shell 2>/dev/null | grep -q "[[:space:]]${bucket}[[:space:]]"; then
		return 0
	fi
	echo "对象存储缺少 bucket ${bucket}，正在创建 ..."
	if ! echo "s3.bucket.create -name ${bucket}" | docker compose "${COMPOSE_ARGS[@]}" exec -T seaweedfs weed shell >/dev/null 2>&1; then
		echo "提示：自动创建 bucket 失败；若使用外部对象存储请忽略，否则请手动创建。" >&2
	fi
}

case " $* " in
	*" --dry-run "*) ;;
	*) ensure_bucket ;;
esac

# SEED_* 口令通过 exec -e 透传（空值 = 使用 cmd/seed 内的 dev 缺省值）。
# 口令只用于本地开发，禁止在部署环境沿用（生产环境 cmd/seed 会直接拒绝启动）。
#
# 素材目录：脚本默认显式传 -assets-dir（容器内 /devdata/seed/media）；用户自己
# 传了 -assets-dir/--assets-dir 时不再注入（用户参数在后，覆盖默认）。默认解析
# 仍保留兜底：容器内直接 go run ./cmd/seed 也会解析到 ../devdata/seed/media。
seed_cmd=(docker compose "${COMPOSE_ARGS[@]}" exec -T
	-e "SEED_QA_PASSWORD=${SEED_QA_PASSWORD:-}"
	-e "SEED_ADMIN_PASSWORD=${SEED_ADMIN_PASSWORD:-}"
	backend go run ./cmd/seed)
case " $* " in
	*" -assets-dir "* | *" -assets-dir="* | *" --assets-dir "* | *" --assets-dir="*) ;;
	*) seed_cmd+=(-assets-dir "$ASSETS_DIR_CONTAINER") ;;
esac
seed_cmd+=("$@")
set +e
"${seed_cmd[@]}"
rc=$?
set -e
[ "$rc" -ne 0 ] && exit "$rc"

case " $* " in
	*" --dry-run "*) exit 0 ;;
esac

# seed 直接写 app_configs（默认浏览锚点/应用配置），而运行中的 backend 进程持有
# AppConfig 内存缓存（appconfig.PersistenceSource 启动时加载）：不重启的话
# client_config 仍下发旧默认位置，附近/地图页与新种子数据对不上（2026-10-29 联调实测）。
# dev 环境重启约 20-40s（air 增量编译 + 迁移检查），这里自动完成并等待 healthy。
echo "重启 backend 使 app_configs（默认浏览锚点等）生效 ..."
if docker compose "${COMPOSE_ARGS[@]}" restart backend >/dev/null 2>&1; then
	health=""
	for _ in $(seq 1 90); do
		container_id="$(docker compose "${COMPOSE_ARGS[@]}" ps -q backend 2>/dev/null || true)"
		health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container_id" 2>/dev/null || true)"
		[ "$health" = "healthy" ] && break
		sleep 2
	done
	if [ "$health" = "healthy" ]; then
		echo "backend 已重启并就绪（AppConfig 已重新加载）。"
	else
		echo "提示：backend 尚未恢复 healthy，请用 docker compose -f docker-compose.yml -f docker-compose.dev.yml ps 检查。" >&2
	fi
else
	echo "提示：backend 重启失败；请手动执行 docker compose -f docker-compose.yml -f docker-compose.dev.yml restart backend 使 AppConfig 生效。" >&2
fi
