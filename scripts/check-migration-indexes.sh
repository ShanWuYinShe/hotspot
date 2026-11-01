#!/usr/bin/env bash
# ── 迁移声明索引 vs 数据库实际索引 一致性校验（只读）─────────────────
# 背景（2026-11-01 线上排查）：存量库首次接入时，schema_migrations 会做一次性
# 「基线登记」——核心表已齐的库只登记迁移文件、不执行 SQL（见
# backend-go/internal/platform/migrate/embedded.go 的 legacy baseline 分支）。
# 若某索引在基线时的数据快照里不存在（快照早于声明它的迁移），它会被永久标记
# 为「已应用」，迁移里的 CREATE INDEX 再也不会执行。线上 82 个声明索引里
# idx_moments_keyset（056 声明）就是这样缺失的（它 2026-10-07 才加入仓库，而
# 线上库 2026-10-08 做基线，快照里没有该索引）。
#
# 本脚本去掉 SQL 行注释后，从 backend-go/migrations/*.sql 提取声明的索引名
# （CREATE [UNIQUE] INDEX [IF NOT EXISTS] <name>，扣除后续 DROP INDEX），与目标
# 库 public schema 的实际索引逐名比对；缺失或 INVALID（并发建索引失败残留）的
# 索引会让退出码为 1。
#
# 用法（默认检查仓库根 compose 栈的 postgres 服务 / hotspot 库；worktree 无
# .env 时自动回落到主工作树的 compose 环境）：
#   ./scripts/check-migration-indexes.sh
#   ./scripts/check-migration-indexes.sh --container hotspot-postgres-1 --database hotspot
#   ./scripts/check-migration-indexes.sh --database hotspot_it
#   ./scripts/check-migration-indexes.sh --dsn "postgres://user:pass@host:5432/db"  # 需本机 psql
# 环境变量：PG_CONTAINER（同 --container，显式 --dsn 优先）、POSTGRES_DSN（无显式
# 参数时作为 --dsn 回退）、PG_USER（默认 postgres）
#
# 只读：仅执行目录查询/系统视图 SELECT，不建索引、不改库；可对新库、存量库、
# 线上库（低峰只读）任选目标执行。
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
MIGRATIONS_DIR="$ROOT/backend-go/migrations"
DATABASE=hotspot
PG_USER="${PG_USER:-postgres}"
CONTAINER=""
DSN=""

usage() { sed -n '2,26p' "$0" | sed -e 's/^# //' -e 's/^#$//'; }

while [ $# -gt 0 ]; do
  case "$1" in
    --container) CONTAINER="$2"; shift 2 ;;
    --database)  DATABASE="$2"; shift 2 ;;
    --dsn)       DSN="$2"; shift 2 ;;
    --user)      PG_USER="$2"; shift 2 ;;
    -h|--help)   usage; exit 0 ;;
    *) echo "未知参数：$1（--help 查看用法）" >&2; exit 2 ;;
  esac
done

# 目标选择：显式 --dsn > 显式 --container > PG_CONTAINER 环境变量 >
# POSTGRES_DSN 环境变量（本机 psql 直连）> 自动探测仓库根 compose 的 postgres
# 服务（worktree 无 .env 时回落主工作树），见 run_query。
if [ -z "$CONTAINER" ] && [ -z "$DSN" ]; then
  if [ -n "${PG_CONTAINER:-}" ]; then
    CONTAINER="$PG_CONTAINER"
  elif [ -n "${POSTGRES_DSN:-}" ]; then
    DSN="$POSTGRES_DSN"
  fi
fi

# 目标库访问：--dsn（本机 psql 直连）> --container（docker exec）> 自动探测
# 仓库根 compose 的 postgres 服务（worktree 无 .env 时回落主工作树）。
detect_container() {
  local compose_root="$ROOT"
  if [ ! -f "$compose_root/.env" ]; then
    local main_wt
    main_wt=$(git -C "$ROOT" worktree list --porcelain 2>/dev/null | awk '/^worktree /{print $2; exit}')
    if [ -n "${main_wt:-}" ] && [ -f "$main_wt/.env" ]; then
      compose_root="$main_wt"
    fi
  fi
  docker compose --project-directory "$compose_root" \
    -f "$compose_root/docker-compose.yml" \
    -f "$compose_root/docker-compose.dev.yml" \
    ps -q postgres 2>/dev/null | head -n 1
}

run_query() {
  local sql="$1"
  if [ -n "$DSN" ]; then
    command -v psql >/dev/null 2>&1 || { echo "!! --dsn 模式需要本机 psql 客户端" >&2; exit 2; }
    psql "$DSN" -At -v ON_ERROR_STOP=1 -c "$sql"
    return
  fi
  local container="$CONTAINER"
  if [ -z "$container" ]; then
    container=$(detect_container)
    if [ -z "$container" ]; then
      echo "!! 没有运行中的 postgres 容器：用 --container / --dsn 指定，或先启动 compose dev 栈" >&2
      exit 2
    fi
  fi
  docker exec -i "$container" psql -U "$PG_USER" -d "$DATABASE" -At -v ON_ERROR_STOP=1 -c "$sql"
}

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# 1) 迁移声明索引：去 -- 行注释 → 多行拼一行 → 提取 CREATE [UNIQUE] INDEX 名。
#    再扣除 DROP INDEX（历史上无，防未来脚本误报）。
declared_all="$TMP/declared_all.txt"
dropped="$TMP/dropped.txt"
declared="$TMP/declared.txt"
for f in "$MIGRATIONS_DIR"/*.sql; do
  sed -e 's/--.*$//' "$f"
done \
  | tr '\n' ' ' \
  | grep -ioE 'create[[:space:]]+(unique[[:space:]]+)?index[[:space:]]+(if[[:space:]]+not[[:space:]]+exists[[:space:]]+)?[a-z0-9_]+' \
  | awk '{ print $NF }' \
  | sort -u > "$declared_all" || true

for f in "$MIGRATIONS_DIR"/*.sql; do
  sed -e 's/--.*$//' "$f"
done \
  | tr '\n' ' ' \
  | grep -ioE 'drop[[:space:]]+index[[:space:]]+(if[[:space:]]+exists[[:space:]]+)?[a-z0-9_]+' \
  | awk '{ print $NF }' \
  | sort -u > "$dropped" || true

comm -23 "$declared_all" "$dropped" > "$declared"

# 2) 目标库实际索引（public schema，含主键/唯一约束隐式索引）与 INVALID 列表。
actual="$TMP/actual.txt"
invalid="$TMP/invalid.txt"
run_query "SELECT indexname FROM pg_indexes WHERE schemaname = 'public' ORDER BY indexname" | sort -u > "$actual"
run_query "SELECT c.relname FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND NOT i.indisvalid ORDER BY 1" | sort -u > "$invalid" || true

# 3) 比对。
missing="$TMP/missing.txt"
declared_invalid="$TMP/declared_invalid.txt"
comm -23 "$declared" "$actual" > "$missing"
comm -12 "$declared" "$invalid" > "$declared_invalid"

declared_count=$(wc -l < "$declared" | tr -d ' ')
actual_count=$(wc -l < "$actual" | tr -d ' ')
missing_count=$(wc -l < "$missing" | tr -d ' ')
invalid_count=$(wc -l < "$declared_invalid" | tr -d ' ')

echo "目标库：${DATABASE}"
echo "迁移声明索引：${declared_count} 个；库中 public schema 实际索引：${actual_count} 个（含主键/唯一约束索引）"

if [ "$missing_count" = "0" ] && [ "$invalid_count" = "0" ]; then
  echo "一致性校验 OK：迁移声明的索引在库中全部存在且 VALID"
  exit 0
fi

if [ "$missing_count" != "0" ]; then
  echo ""
  echo "!! 缺失索引（迁移声明但库中不存在）：${missing_count} 个"
  sed 's/^/   - /' "$missing"
  echo ""
  echo "   常见原因：存量库基线（schema_migrations 只登记不执行）跳过了声明该索引"
  echo "   的迁移文件。修法：新增幂等迁移重放 CREATE INDEX IF NOT EXISTS（参照"
  echo "   058_moments_keyset_index_backfill.sql），不要直接改历史迁移文件（已登记"
  echo "   的文件不会再执行）。"
fi

if [ "$invalid_count" != "0" ]; then
  echo ""
  echo "!! INVALID 索引（并发建索引失败残留，查询不会使用）：${invalid_count} 个"
  sed 's/^/   - /' "$declared_invalid"
  echo "   处理：DROP INDEX 后重建（迁移内禁止 CONCURRENTLY，见 skill hotspot-backend-migrations）。"
fi

exit 1
