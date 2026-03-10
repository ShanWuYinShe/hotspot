#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"

# shellcheck source=/dev/null
source "$ROOT_DIR/deploy/config.sh"

if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose -f "$COMPOSE_FILE")
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD=(docker-compose -f "$COMPOSE_FILE")
else
    echo "未检测到 docker compose 或 docker-compose，无法准备 Docker 编译环境"
    exit 1
fi

if [ ! -f "$BACKEND_DIR/Cargo.toml" ]; then
    echo "backend/Cargo.toml 不存在，无法执行 SQLx prepare"
    exit 1
fi

echo "Preparing Docker-backed SQLx environment..."

ensure_postgres_database() {
    local database_name="$1"
    local exists
    exists=$("${COMPOSE_CMD[@]}" exec -T postgres psql -U "$DOCKER_DB_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${database_name}'" | tr -d ' \n' || true)
    if [ "$exists" = "1" ]; then
        return
    fi
    echo "Creating database ${database_name}..."
    "${COMPOSE_CMD[@]}" exec -T postgres psql -U "$DOCKER_DB_USER" -d postgres -c "CREATE DATABASE ${database_name};" >/dev/null
}

if ! command -v sqlx >/dev/null 2>&1; then
    echo "sqlx-cli 未安装，开始安装..."
    cargo install sqlx-cli --no-default-features --features postgres
fi

echo "Starting PostgreSQL and test Redis containers..."
"${COMPOSE_CMD[@]}" up -d postgres redis-test

echo "Waiting for PostgreSQL to become ready..."
for _ in $(seq 1 30); do
    if "${COMPOSE_CMD[@]}" exec -T postgres pg_isready -U "$DOCKER_DB_USER" -d "$DOCKER_DB_NAME" >/dev/null 2>&1; then
        break
    fi
    sleep 2
done

if ! "${COMPOSE_CMD[@]}" exec -T postgres pg_isready -U "$DOCKER_DB_USER" -d "$DOCKER_DB_NAME" >/dev/null 2>&1; then
    echo "PostgreSQL 未在预期时间内就绪"
    exit 1
fi

ensure_postgres_database "$DOCKER_DB_NAME"
ensure_postgres_database "$TEST_DB_NAME"

cd "$BACKEND_DIR"

echo "Running migrations against runtime database..."
DATABASE_URL="$DOCKER_DATABASE_URL" cargo sqlx migrate run

echo "Running migrations against test database..."
DATABASE_URL="$TEST_DATABASE_URL" cargo sqlx migrate run

echo "Generating SQLx offline metadata..."
DATABASE_URL="$DOCKER_DATABASE_URL" cargo sqlx prepare --workspace

echo "Preparation complete. Recommended follow-up commands:"
echo "  cd backend"
echo "  DATABASE_URL=\"$DOCKER_DATABASE_URL\" cargo check"
echo "  DATABASE_URL=\"$TEST_DATABASE_URL\" REDIS_URL=\"$TEST_REDIS_URL\" TEST_REDIS_URL=\"$TEST_REDIS_URL\" cargo test"
echo "  ${COMPOSE_CMD[*]} up --build"
