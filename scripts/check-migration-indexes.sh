#!/usr/bin/env bash
# ── 期望索引 vs 数据库实际索引 一致性校验（只读）────────────────────
# 期望索引来源：backend-go/migrations/schema_manifest.json（由「在全新库执行全部
# 迁移」生成，是迁移集合真值的快照；已包含 DROP TABLE/INDEX 之后的净结果）。
# 与目标库 public schema 的实际索引逐名比对；缺失或 INVALID 索引会让退出码为 1。
#
# 历史背景：旧 runner 的「存量库基线（只登记不执行）」曾让线上永久缺失
# idx_moments_keyset（056 声明）。新 runner 已删除启发式基线，并在启动/CI 用
# schema manifest 校验期望对象；本脚本保留为独立的只读诊断工具（可对任意库运行）。
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

# 1) 期望索引：从 schema_manifest.json 提取 indexes（格式 "table:indexname"）。
MANIFEST="$MIGRATIONS_DIR/schema_manifest.json"
declared_all="$TMP/declared_all.txt"
declared="$TMP/declared.txt"
if [ ! -f "$MANIFEST" ]; then
  echo "!! 缺少 ${MANIFEST}：用 go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json 生成" >&2
  exit 2
fi
grep -oE '"[^"]+:[^"]+"' "$MANIFEST" | tr -d '"' | awk -F: '{print $NF}' | sort -u > "$declared_all"
cp "$declared_all" "$declared"

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
echo "期望索引（schema manifest）：${declared_count} 个；库中 public schema 实际索引：${actual_count} 个（含主键/唯一约束索引）"

if [ "$missing_count" = "0" ] && [ "$invalid_count" = "0" ]; then
  echo "一致性校验 OK：迁移声明的索引在库中全部存在且 VALID"
  exit 0
fi

if [ "$missing_count" != "0" ]; then
  echo ""
  echo "!! 缺失索引（manifest 期望但库中不存在）：${missing_count} 个"
  sed 's/^/   - /' "$missing"
  echo ""
  echo "   常见原因：目标库 schema 落后于迁移集合（未执行 pending 迁移，或手工改过库）。"
  echo "   修法：先 go run ./cmd/migrate 应用 pending；仍有缺失说明版本记录与 schema 漂移，"
  echo "   执行 go run ./cmd/migrate -verify-only 查看完整清单后按需修复（不要改历史迁移文件）。"
fi

if [ "$invalid_count" != "0" ]; then
  echo ""
  echo "!! INVALID 索引（并发建索引失败残留，查询不会使用）：${invalid_count} 个"
  sed 's/^/   - /' "$declared_invalid"
  echo "   处理：DROP INDEX 后重建（迁移内禁止 CONCURRENTLY，见 skill hotspot-backend-migrations）。"
fi

exit 1
