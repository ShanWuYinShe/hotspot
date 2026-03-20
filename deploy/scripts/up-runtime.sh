#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.runtime.yml"
CERT_DIR="$ROOT/tmp/certs"
ENV_NAME="${HOTSPOT_ENV_NAME:-default}"
PROJECT_NAME=""
HTTP_PORT="${HOTSPOT_RUNTIME_HTTP_PORT:-8088}"
HTTPS_PORT="${HOTSPOT_RUNTIME_HTTPS_PORT:-8443}"
PUBLIC_ORIGIN="${HOTSPOT_PUBLIC_ORIGIN:-https://localhost:${HTTPS_PORT}}"
CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-$CERT_DIR/localhost.crt}"
KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-$CERT_DIR/localhost.key}"
APP_SERVICES=(backend frontend nginx)
DATA_SERVICES=(postgres redis minio)
SERVICES=("${APP_SERVICES[@]}")
RECREATE_SERVICES=()
CLEAN_MODE="none"
CLEAN_TARGETS=()

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
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
Usage: ./deploy/scripts/up-runtime.sh [options]

Options:
  --env-name NAME     Isolated environment name (default: default)
  --http-port PORT    Public HTTP port for nginx redirect entry (default: 8088)
  --https-port PORT   Public HTTPS port for nginx main entry (default: 8443)
  --services a,b,c    Start only the specified services (default: backend,frontend,nginx)
  --recreate a,b,c    Force recreate specific services before start
  --clean-all         Remove all runtime containers and data volumes before start
  --clean a,b,c       Remove selected data services before start (postgres,redis,minio)
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
    --http-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --http-port" >&2; exit 1; }
      HTTP_PORT="$1"
      ;;
    --https-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --https-port" >&2; exit 1; }
      HTTPS_PORT="$1"
      ;;
    --services)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --services" >&2; exit 1; }
      IFS=',' read -r -a SERVICES <<< "$1"
      ;;
    --recreate)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --recreate" >&2; exit 1; }
      IFS=',' read -r -a RECREATE_SERVICES <<< "$1"
      ;;
    --clean-all)
      CLEAN_MODE="all"
      ;;
    --clean)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --clean" >&2; exit 1; }
      CLEAN_MODE="partial"
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
PUBLIC_ORIGIN="${HOTSPOT_PUBLIC_ORIGIN:-https://localhost:${HTTPS_PORT}}"

if [[ ! -f "$CERT_PATH" || ! -f "$KEY_PATH" ]]; then
  cat >&2 <<EOF
TLS certificate or key not found.
Expected:
  HOTSPOT_TLS_CERT_PATH=$CERT_PATH
  HOTSPOT_TLS_KEY_PATH=$KEY_PATH

For Cloudflare Full mode, you can also use a Cloudflare Origin Certificate here.
For local testing, you can generate certs with mkcert, for example:
  mkdir -p "$CERT_DIR"
  mkcert -cert-file "$CERT_PATH" -key-file "$KEY_PATH" localhost 127.0.0.1 ::1
EOF
  exit 1
fi

for service in "${SERVICES[@]}"; do
  contains "$service" backend frontend nginx postgres redis minio createbuckets || {
    echo "unsupported service in --services: $service" >&2
    exit 1
  }
done

for service in "${RECREATE_SERVICES[@]}"; do
  contains "$service" backend frontend nginx postgres redis minio createbuckets || {
    echo "unsupported service in --recreate: $service" >&2
    exit 1
  }
done

if [[ "$CLEAN_MODE" == "partial" ]]; then
  for target in "${CLEAN_TARGETS[@]}"; do
    contains "$target" "${DATA_SERVICES[@]}" || {
      echo "unsupported service in --clean: $target" >&2
      exit 1
    }
  done
fi

export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"
export HOTSPOT_PUBLIC_ORIGIN="$PUBLIC_ORIGIN"
export HOTSPOT_RUNTIME_HTTP_PORT="$HTTP_PORT"
export HOTSPOT_RUNTIME_HTTPS_PORT="$HTTPS_PORT"

cd "$ROOT"

if [[ "$CLEAN_MODE" == "all" ]]; then
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
elif [[ "$CLEAN_MODE" == "partial" ]]; then
  if [[ ${#CLEAN_TARGETS[@]} -gt 0 ]]; then
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" stop "${CLEAN_TARGETS[@]}" >/dev/null 2>&1 || true
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" rm -f -s -v "${CLEAN_TARGETS[@]}" >/dev/null 2>&1 || true
    for target in "${CLEAN_TARGETS[@]}"; do
      case "$target" in
        postgres) docker volume rm -f "${PROJECT_NAME}_runtime_postgres_data" >/dev/null 2>&1 || true ;;
        redis) docker volume rm -f "${PROJECT_NAME}_runtime_redis_data" >/dev/null 2>&1 || true ;;
        minio) docker volume rm -f "${PROJECT_NAME}_runtime_minio_data" >/dev/null 2>&1 || true ;;
      esac
    done
  fi
fi

BASE_SERVICES=("${SERVICES[@]}")
if contains backend "${SERVICES[@]}" || contains frontend "${SERVICES[@]}" || contains nginx "${SERVICES[@]}"; then
  for dep in postgres redis minio createbuckets; do
    if ! contains "$dep" "${BASE_SERVICES[@]}"; then
      BASE_SERVICES+=("$dep")
    fi
  done
fi

UP_ARGS=(up -d --build)
if [[ ${#RECREATE_SERVICES[@]} -gt 0 ]]; then
  UP_ARGS+=(--force-recreate)
fi
UP_ARGS+=("${BASE_SERVICES[@]}")

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "${UP_ARGS[@]}"

echo "HotSpot runtime is starting."
echo "Environment name: $ENV_NAME"
echo "Compose project: $PROJECT_NAME"
echo "HTTPS entrypoint: $PUBLIC_ORIGIN"
echo "HTTP redirect port:  http://localhost:${HTTP_PORT}"
echo "Started services: $(join_by ',' "${BASE_SERVICES[@]}")"
if [[ "$CLEAN_MODE" == "all" ]]; then
  echo "Data reset: all runtime data volumes were removed before start."
elif [[ "$CLEAN_MODE" == "partial" && ${#CLEAN_TARGETS[@]} -gt 0 ]]; then
  echo "Data reset: $(join_by ',' "${CLEAN_TARGETS[@]}")"
else
  echo "Data reset: none (existing postgres/redis/minio data preserved)."
fi
if [[ ${#RECREATE_SERVICES[@]} -gt 0 ]]; then
  echo "Forced recreate: $(join_by ',' "${RECREATE_SERVICES[@]}")"
fi
