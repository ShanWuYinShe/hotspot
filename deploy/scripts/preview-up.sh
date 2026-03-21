#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/preview.yml"
CERT_DIR="$ROOT/tmp/certs"
PROJECT_NAME="$(preview_project_name)"
HTTP_PORT="8088"
HTTPS_PORT="8443"
HTTP_PORT_EXPLICIT=0
HTTPS_PORT_EXPLICIT=0
PUBLIC_ORIGIN=""
CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-$CERT_DIR/localhost.crt}"
KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-$CERT_DIR/localhost.key}"

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/preview-up.sh [options]

Options:
  --http-port PORT    Public HTTP port for nginx redirect entry (default: 8088)
  --https-port PORT   Public HTTPS port for nginx main entry (default: 8443)
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --http-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --http-port" >&2; exit 1; }
      HTTP_PORT="$1"
      HTTP_PORT_EXPLICIT=1
      ;;
    --https-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --https-port" >&2; exit 1; }
      HTTPS_PORT="$1"
      HTTPS_PORT_EXPLICIT=1
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

PUBLIC_ORIGIN="${HOTSPOT_PREVIEW_PUBLIC_ORIGIN:-https://localhost:${HTTPS_PORT}}"

if [[ ! -f "$CERT_PATH" || ! -f "$KEY_PATH" ]]; then
  cat >&2 <<EOF
TLS certificate or key not found.
Expected:
  HOTSPOT_TLS_CERT_PATH=$CERT_PATH
  HOTSPOT_TLS_KEY_PATH=$KEY_PATH

For local testing, you can generate certs with mkcert, for example:
  mkdir -p "$CERT_DIR"
  mkcert -cert-file "$CERT_PATH" -key-file "$KEY_PATH" localhost 127.0.0.1 ::1
EOF
  exit 1
fi

if ! docker image inspect "$(preview_backend_image)" >/dev/null 2>&1 || ! docker image inspect "$(preview_frontend_image)" >/dev/null 2>&1; then
  echo "preview images are missing. Run ./deploy/scripts/preview-build.sh first." >&2
  exit 1
fi

if [[ $HTTP_PORT_EXPLICIT -eq 0 ]]; then
  if port_in_use "$HTTP_PORT"; then
    echo "default preview HTTP port $HTTP_PORT is already in use; pass --http-port to override" >&2
    exit 1
  fi
elif port_in_use "$HTTP_PORT"; then
  echo "preview HTTP port $HTTP_PORT is already in use" >&2
  exit 1
fi
if [[ $HTTPS_PORT_EXPLICIT -eq 0 ]]; then
  if port_in_use "$HTTPS_PORT"; then
    echo "default preview HTTPS port $HTTPS_PORT is already in use; pass --https-port to override" >&2
    exit 1
  fi
elif port_in_use "$HTTPS_PORT"; then
  echo "preview HTTPS port $HTTPS_PORT is already in use" >&2
  exit 1
fi

export HOTSPOT_PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
export HOTSPOT_PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
export HOTSPOT_PREVIEW_HTTP_PORT="$HTTP_PORT"
export HOTSPOT_PREVIEW_HTTPS_PORT="$HTTPS_PORT"
export HOTSPOT_PREVIEW_PUBLIC_ORIGIN="$PUBLIC_ORIGIN"
export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"

cd "$ROOT"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d

echo "Preview environment is running."
echo "Compose project: $PROJECT_NAME"
echo "HTTPS entrypoint: $PUBLIC_ORIGIN"
echo "HTTP redirect entrypoint: http://localhost:${HTTP_PORT}"
