#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT/deploy/docker-compose.runtime.yml"
ENV_NAME="${HOTSPOT_ENV_NAME:-default}"
PROJECT_NAME=""
HTTPS_PORT="${HOTSPOT_RUNTIME_HTTPS_PORT:-8443}"
HOSTNAME="${HOTSPOT_RUNTIME_HOST:-localhost}"
CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"
SKIP_HOME=0
SKIP_HEALTH=0
SKIP_S3=0

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/check-runtime.sh [options]

Options:
  --env-name NAME     Target isolated environment name (default: default)
  --https-port PORT   HTTPS port to check (default: 8443)
  --host HOST         Hostname to check (default: localhost)
  --skip-home         Skip checking GET /
  --skip-health       Skip checking POST /api/v1/system/health
  --skip-s3           Skip checking GET /s3/
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
    --https-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --https-port" >&2; exit 1; }
      HTTPS_PORT="$1"
      ;;
    --host)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --host" >&2; exit 1; }
      HOSTNAME="$1"
      ;;
    --skip-home)
      SKIP_HOME=1
      ;;
    --skip-health)
      SKIP_HEALTH=1
      ;;
    --skip-s3)
      SKIP_S3=1
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
BASE_URL="https://${HOSTNAME}:${HTTPS_PORT}"

export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"

cd "$ROOT"

echo "Environment name: $ENV_NAME"
echo "Compose project: $PROJECT_NAME"
echo "Checking base URL: $BASE_URL"

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" ps

if [[ $SKIP_HOME -eq 0 ]]; then
  echo
  echo "[check] GET /"
  curl -k -f -sS "$BASE_URL/" >/dev/null
  echo "ok"
fi

if [[ $SKIP_HEALTH -eq 0 ]]; then
  echo
  echo "[check] POST /api/v1/system/health"
  curl -k -f -sS -X POST "$BASE_URL/api/v1/system/health" -H "Content-Type: application/json" -d "{}"
  echo
fi

if [[ $SKIP_S3 -eq 0 ]]; then
  echo
  echo "[check] GET /s3/"
  curl -k -sS -o /tmp/hotspot-s3-check.out -w "%{http_code}" "$BASE_URL/s3/" >/tmp/hotspot-s3-check.code
  code="$(cat /tmp/hotspot-s3-check.code)"
  if [[ "$code" == "200" || "$code" == "403" || "$code" == "404" ]]; then
    echo "ok (status $code)"
  else
    echo "unexpected status $code" >&2
    exit 1
  fi
fi

echo

echo "Runtime checks passed."
