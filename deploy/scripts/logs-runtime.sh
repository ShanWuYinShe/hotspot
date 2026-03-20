#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.runtime.yml"
source "$ROOT/deploy/scripts/common.sh"
ENV_NAME="$(detect_env_name "$ROOT")"
PROJECT_NAME=""
TAIL_LINES="200"
FOLLOW=0
SERVICES=()
CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"

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

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/logs-runtime.sh [options]

Options:
  --env-name NAME     Target isolated environment name (default: default)
  --services a,b,c    Show logs only for specified services
  --tail N            Number of lines to show (default: 200)
  --follow            Follow logs continuously
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
    --tail)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --tail" >&2; exit 1; }
      TAIL_LINES="$1"
      ;;
    --follow)
      FOLLOW=1
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

export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"

cd "$ROOT"

ARGS=(logs --tail "$TAIL_LINES")
if [[ $FOLLOW -eq 1 ]]; then
  ARGS+=(--follow)
fi
if [[ ${#SERVICES[@]} -gt 0 ]]; then
  ARGS+=("${SERVICES[@]}")
fi

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "${ARGS[@]}"
