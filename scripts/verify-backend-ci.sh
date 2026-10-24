#!/usr/bin/env bash
# 本地复跑 GitHub CI 的 backend 门禁（与 .github/workflows/ci.yml 的 backend /
# backend-integration 两个 job 对齐）。
#
# 背景（2026-10-23）：多次提交只在本地跑 go build/vet/test -short 就推送，CI 的
# 真实库集成测试（TestPostgresIT）与 -race/覆盖率门禁未覆盖，导致 main 连续红灯。
# 该脚本把 CI 检查原样搬下来，推送前跑一遍即可避免同类问题。
#
# 用法：
#   ./scripts/verify-backend-ci.sh              # 静态检查 + 单测(race/覆盖率门禁)
#   ./scripts/verify-backend-ci.sh --with-db    # 追加真实库仓库集成测试(TestPostgresIT)
#   ./scripts/verify-backend-ci.sh --with-e2e   # 追加 HTTP 全链路集成测试
#   ./scripts/verify-backend-ci.sh --all        # 全部（含 govulncheck）
#
# 依赖：go、gofmt、awk；--with-db/--with-e2e 需要运行中的 docker compose dev 栈。
# 在 worktree 内执行时，DB/E2E 阶段回落到主工作树的 compose/env（验证主树源码）。
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
BACKEND="$ROOT/backend-go"
WITH_DB=0; WITH_E2E=0; WITH_VULN=0
for arg in "$@"; do
  case "$arg" in
    --with-db) WITH_DB=1 ;;
    --with-e2e) WITH_E2E=1 ;;
    --with-vuln) WITH_VULN=1 ;;
    --all) WITH_DB=1; WITH_E2E=1; WITH_VULN=1 ;;
    -h|--help) sed -n "2,20p" "$0"; exit 0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

# compose 相关阶段需要 .env 凭证与运行中的栈；worktree 内回落到主工作树。
COMPOSE_ROOT="$ROOT"
if [ ! -f "$ROOT/.env" ]; then
  main_wt=$(git -C "$ROOT" worktree list --porcelain 2>/dev/null | awk "/^worktree /{print \$2; exit}")
  if [ -n "${main_wt:-}" ] && [ -f "$main_wt/.env" ]; then COMPOSE_ROOT="$main_wt"; fi
fi
COMPOSE=(docker compose --project-directory "$COMPOSE_ROOT" -f "$COMPOSE_ROOT/docker-compose.yml" -f "$COMPOSE_ROOT/docker-compose.dev.yml")

step() { printf "\n\033[36;1m==> %s\033[0m\n" "$*"; }
die()  { printf "\033[31;1mFAILED: %s\033[0m\n" "$*" >&2; exit 1; }

step "go build (cmd/server)"
go build -C "$BACKEND" -o /dev/null ./cmd/server

step "gofmt"
UNFORMATTED=$(gofmt -l "$BACKEND")
if [ -n "$UNFORMATTED" ]; then
  echo "以下文件未通过 gofmt（执行 gofmt -w backend-go 修复）:"; echo "$UNFORMATTED"; die "gofmt"
fi

step "go vet"
go vet -C "$BACKEND" ./...

if [ "$WITH_VULN" = "1" ]; then
  step "govulncheck (CI 固定 v1.8.0)"
  if ! command -v govulncheck >/dev/null 2>&1; then go install golang.org/x/vuln/cmd/govulncheck@v1.8.0; fi
  (cd "$BACKEND" && "$(go env GOPATH)/bin/govulncheck" ./...)
fi

step "check migration numbering"
"$ROOT/scripts/check-migrations.sh"

step "go test -race -cover（internal/modules 覆盖率 >=70% 门禁）"
LOG=$(mktemp -t gotest.XXXXXX.log)
cleanup_log() { rm -f "$LOG"; }
trap cleanup_log EXIT
( cd "$BACKEND" && go test -race -count=1 -cover $(go list ./... | grep -v /test/integration) 2>&1 | tee "$LOG" )
awk '/^ok .*coverage: [0-9.]+% of statements/ {
  pkg = $2; cov = $0;
  sub(/^.*coverage: /, "", cov); sub(/% of statements.*$/, "", cov);
  if (pkg ~ /modules\// && cov + 0 < 70.0) { print "COVERAGE BELOW 70%: " pkg " " cov "%"; failed = 1 }
} END { if (failed) exit 1 }' "$LOG" || die "模块覆盖率低于 70%"

PG_PASSWORD=$(grep -E "^POSTGRES_PASSWORD=" "$COMPOSE_ROOT/.env" 2>/dev/null | head -1 | cut -d= -f2- || true)
PG_PASSWORD=${PG_PASSWORD:-postgres}

# 本地测试用 JWT 密钥：运行时随机生成（>=32 字节），不在仓库写任何看起来像
# 凭据的字面量——gitleaks 会对含 secret/key 词根的长字面量告警（CI Secret scan
# 全历史扫描，2026-10-23 实测踩到）。可用环境变量 AUTH_TOKEN_SECRET 覆盖。
VERIFY_AUTH_SECRET=${AUTH_TOKEN_SECRET:-$(printf 'verify-local-%s' "$(openssl rand -hex 24)")}

if [ "$WITH_DB" = "1" ]; then
  step "真实库仓库集成测试（TestPostgresIT + 双实现契约测试，库 hotspot_it）"
  DB=hotspot_it
  "${COMPOSE[@]}" exec -T postgres psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB'" | grep -q 1 \
    || "${COMPOSE[@]}" exec -T postgres psql -U postgres -c "CREATE DATABASE $DB"
  DSN="postgres://postgres:$PG_PASSWORD@localhost:5432/$DB?sslmode=disable"
  ( cd "$BACKEND" && POSTGRES_DSN="$DSN" APP_ENV=ci AUTH_TOKEN_SECRET="$VERIFY_AUTH_SECRET" go run ./cmd/migrate >/dev/null )
  ( cd "$BACKEND" && POSTGRES_DSN="$DSN" APP_ENV=ci AUTH_TOKEN_SECRET="$VERIFY_AUTH_SECRET" go test -count=1 -run "TestPostgresIT|Contract" ./internal/modules/... )
fi

if [ "$WITH_E2E" = "1" ]; then
  step "HTTP 全链路集成测试（test/integration）"
  "${COMPOSE[@]}" --profile test up -d backend-test >/dev/null
  cleanup_e2e() { "${COMPOSE[@]}" --profile test stop backend-test >/dev/null 2>&1 || true; cleanup_log; }
  trap cleanup_e2e EXIT
  READY=0
  for _ in $(seq 1 60); do
    if curl -sf -X POST -H "Content-Type: application/json" -d "{}" http://localhost:8082/public/v1/system/health | grep -q '"code":0'; then READY=1; break; fi
    sleep 3
  done
  [ "$READY" = "1" ] || die "backend-test 未就绪（docker compose logs backend-test 查看）"
  ( cd "$BACKEND" && BACKEND_URL=http://localhost:8082 \
      POSTGRES_DSN="postgres://postgres:$PG_PASSWORD@localhost:5432/hotspot_test?sslmode=disable" \
      NGINX_PORT=${NGINX_PORT:-8888} APP_ENV=test AUTH_TOKEN_SECRET=${AUTH_TOKEN_SECRET:-dev-secret-change-me} \
      go test -count=1 ./test/integration )
fi

printf "\n\033[32;1mALL CHECKS PASSED\033[0m\n"