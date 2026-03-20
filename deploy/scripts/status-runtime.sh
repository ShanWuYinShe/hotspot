#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.runtime.yml"
source "$ROOT/deploy/scripts/common.sh"
ENV_NAME="$(detect_env_name "$ROOT")"
PROJECT_NAME=""
SHOW_VOLUMES=1
CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/status-runtime.sh [options]

Options:
  --env-name NAME     Target isolated environment name (default: default)
  --no-volumes        Skip volume inspection output
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
    --no-volumes)
      SHOW_VOLUMES=0
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

export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"

cd "$ROOT"

echo "Environment name: $ENV_NAME"
echo "Compose project: $PROJECT_NAME"
echo

echo "Containers:"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" ps || true

if [[ $SHOW_VOLUMES -eq 1 ]]; then
  echo
  echo "Volumes:"
  for volume in runtime_postgres_data runtime_redis_data runtime_minio_data; do
    full_name="${PROJECT_NAME}_${volume}"
    if docker volume inspect "$full_name" >/dev/null 2>&1; then
      mountpoint="$(docker volume inspect "$full_name" --format '{{ .Mountpoint }}')"
      echo "- $full_name"
      echo "  mountpoint: $mountpoint"
    else
      echo "- $full_name (missing)"
    fi
  done
fi
