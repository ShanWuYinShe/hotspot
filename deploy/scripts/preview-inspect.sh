#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/preview.yml"
PROJECT_NAME="$(preview_project_name)"
HTTPS_PORT="8443"
HOSTNAME="localhost"
TAIL_LINES="300"
FORCE_LOGS=0
SKIP_HOME=0
SKIP_HEALTH=0
SKIP_S3=0

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/preview-inspect.sh [options]

Options:
  --https-port PORT   HTTPS port to check (default: 8443)
  --host HOST         Hostname to check (default: localhost)
  --tail N            Number of log lines to show if logs are needed (default: 300)
  --skip-home         Skip checking GET /
  --skip-health       Skip checking POST /api/v1/system/health
  --skip-s3           Skip checking GET /s3/
  --logs              Always show logs, even if checks pass
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
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
    --tail)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --tail" >&2; exit 1; }
      TAIL_LINES="$1"
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
    --logs)
      FORCE_LOGS=1
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
export HOTSPOT_TLS_CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
export HOTSPOT_TLS_KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"

BASE_URL="https://${HOSTNAME}:${HTTPS_PORT}"
cd "$ROOT"

echo "Compose project: $PROJECT_NAME"
echo "Backend image: $HOTSPOT_PREVIEW_BACKEND_IMAGE"
echo "Frontend image: $HOTSPOT_PREVIEW_FRONTEND_IMAGE"
echo

echo "[preview] status"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" ps || true

run_logs() {
  echo
  echo "[preview] logs"
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" logs --tail "$TAIL_LINES"
}

echo
echo "[preview] checks"
if [[ $SKIP_HOME -eq 0 ]]; then
  echo "GET /"
  curl -k -f -sS "$BASE_URL/" >/dev/null
  echo "ok"
fi

if [[ $SKIP_HEALTH -eq 0 ]]; then
  echo
  echo "POST /api/v1/system/health"
  curl -k -f -sS -X POST "$BASE_URL/api/v1/system/health" -H "Content-Type: application/json" -d "{}"
  echo
fi

if [[ $SKIP_S3 -eq 0 ]]; then
  echo
  echo "GET /s3/"
  s3_out="$(mktemp)"
  s3_code="$(mktemp)"
  trap 'rm -f "$s3_out" "$s3_code"' RETURN
  curl -k -sS -o "$s3_out" -w "%{http_code}" "$BASE_URL/s3/" >"$s3_code"
  code="$(cat "$s3_code")"
  if [[ "$code" == "200" || "$code" == "403" || "$code" == "404" ]]; then
    echo "ok (status $code)"
  else
    echo "unexpected status $code" >&2
    run_logs
    exit 1
  fi
fi

if [[ $FORCE_LOGS -eq 1 ]]; then
  run_logs
fi

echo
echo "Preview inspection passed."
