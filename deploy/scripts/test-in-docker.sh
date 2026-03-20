#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.test.yml"
ENV_NAME="${HOTSPOT_ENV_NAME:-default}"
PROJECT_NAME=""

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/test-in-docker.sh [options]

Options:
  --env-name NAME    Isolated environment name for this test run (default: default)
  -h, --help         Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-name)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --env-name" >&2; exit 1; }
      ENV_NAME="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

ENV_NAME="$(sanitize_name "$ENV_NAME")"
PROJECT_NAME="hotspot-test-${ENV_NAME}"

cleanup() {
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

cd "$ROOT"

cleanup

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d --build postgres-test redis-test

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm backend-go-test

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm frontend-validate

echo "Docker test flow passed for environment: $ENV_NAME"
