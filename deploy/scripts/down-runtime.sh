#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.runtime.yml"
ENV_NAME="${HOTSPOT_ENV_NAME:-default}"
PROJECT_NAME=""
MODE="down"
SERVICES=()
CLEAN_TARGETS=()

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

contains() {
  local needle="$1"
  shift
  for item in "$@"; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

join_by() {
  local delimiter="$1"
  shift
  local first=1
  for item in "$@"; do
    if [[ $first -eq 1 ]]; then
      printf '%s' "$item"
      first=0
    else
      printf '%s%s' "$delimiter" "$item"
    fi
  done
}

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/down-runtime.sh [options]

Options:
  --env-name NAME     Target isolated environment name (default: default)
  --services a,b,c    Stop/remove only the specified services
  --remove            Remove containers for the selected services
  --clean-all         Remove all containers and all data volumes for this environment
  --clean a,b,c       Remove selected data volumes (postgres,redis,minio)
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-name)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --env-name" >&2; exit 1; }
      ENV_NAME="$1"
      ;;
    --services)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --services" >&2; exit 1; }
      IFS=',' read -r -a SERVICES <<< "$1"
      ;;
    --remove)
      MODE="remove"
      ;;
    --clean-all)
      MODE="clean-all"
      ;;
    --clean)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --clean" >&2; exit 1; }
      MODE="clean-partial"
      IFS=',' read -r -a CLEAN_TARGETS <<< "$1"
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
PROJECT_NAME="hotspot-runtime-${ENV_NAME}"

for service in "${SERVICES[@]}"; do
  contains "$service" backend frontend nginx postgres redis minio createbuckets || {
    echo "unsupported service in --services: $service" >&2
    exit 1
  }
done

if [[ "$MODE" == "clean-partial" ]]; then
  for target in "${CLEAN_TARGETS[@]}"; do
    contains "$target" postgres redis minio || {
      echo "unsupported service in --clean: $target" >&2
      exit 1
    }
  done
fi

cd "$ROOT"

case "$MODE" in
  down)
    if [[ ${#SERVICES[@]} -gt 0 ]]; then
      docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" stop "${SERVICES[@]}"
      echo "Stopped services for environment $ENV_NAME: $(join_by ',' "${SERVICES[@]}")"
    else
      docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" stop
      echo "Stopped all services for environment $ENV_NAME"
    fi
    ;;
  remove)
    if [[ ${#SERVICES[@]} -gt 0 ]]; then
      docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" rm -f -s "${SERVICES[@]}"
      echo "Removed services for environment $ENV_NAME: $(join_by ',' "${SERVICES[@]}")"
    else
      docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down --remove-orphans
      echo "Removed all containers for environment $ENV_NAME (data preserved)"
    fi
    ;;
  clean-all)
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans
    echo "Removed all containers and data for environment $ENV_NAME"
    ;;
  clean-partial)
    if [[ ${#CLEAN_TARGETS[@]} -eq 0 ]]; then
      echo "no targets provided for --clean" >&2
      exit 1
    fi
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" stop "${CLEAN_TARGETS[@]}" >/dev/null 2>&1 || true
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" rm -f -s -v "${CLEAN_TARGETS[@]}" >/dev/null 2>&1 || true
    for target in "${CLEAN_TARGETS[@]}"; do
      case "$target" in
        postgres) docker volume rm -f "${PROJECT_NAME}_runtime_postgres_data" >/dev/null 2>&1 || true ;;
        redis) docker volume rm -f "${PROJECT_NAME}_runtime_redis_data" >/dev/null 2>&1 || true ;;
        minio) docker volume rm -f "${PROJECT_NAME}_runtime_minio_data" >/dev/null 2>&1 || true ;;
      esac
    done
    echo "Removed data for environment $ENV_NAME: $(join_by ',' "${CLEAN_TARGETS[@]}")"
    ;;
esac
