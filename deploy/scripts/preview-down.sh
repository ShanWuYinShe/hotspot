#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/preview.yml"
PROJECT_NAME="$(preview_project_name)"
MODE="down"
CLEAN_TARGETS=()

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/preview-down.sh [options]

Options:
  --remove            Remove preview containers but preserve data
  --clean-all         Remove preview containers, data volumes, and preview images
  --clean a,b,c       Remove selected data volumes (postgres,redis,minio)
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
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

export HOTSPOT_PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
export HOTSPOT_PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
export HOTSPOT_PREVIEW_HTTP_PORT="${HOTSPOT_PREVIEW_HTTP_PORT:-8088}"
export HOTSPOT_PREVIEW_HTTPS_PORT="${HOTSPOT_PREVIEW_HTTPS_PORT:-8443}"
export HOTSPOT_TLS_CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
export HOTSPOT_TLS_KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"

cd "$ROOT"

case "$MODE" in
  down)
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" stop
    echo "Stopped preview environment"
    ;;
  remove)
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down --remove-orphans
    echo "Removed preview containers (data preserved)"
    ;;
  clean-all)
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans
    docker image rm -f "$(preview_backend_image)" >/dev/null 2>&1 || true
    docker image rm -f "$(preview_frontend_image)" >/dev/null 2>&1 || true
    echo "Removed preview containers, data, and preview images"
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
        postgres) docker volume rm -f "${PROJECT_NAME}_preview_postgres_data" >/dev/null 2>&1 || true ;;
        redis) docker volume rm -f "${PROJECT_NAME}_preview_redis_data" >/dev/null 2>&1 || true ;;
        minio) docker volume rm -f "${PROJECT_NAME}_preview_minio_data" >/dev/null 2>&1 || true ;;
        *) echo "unsupported service in --clean: $target" >&2; exit 1 ;;
      esac
    done
    echo "Removed preview data: $(join_by ',' "${CLEAN_TARGETS[@]}")"
    ;;
esac
