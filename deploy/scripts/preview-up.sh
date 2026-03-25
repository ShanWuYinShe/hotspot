#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/preview.yml"
PRIMARY_ROOT="$(cd "$(git -C "$ROOT" rev-parse --path-format=absolute --git-common-dir 2>/dev/null)/.." 2>/dev/null && pwd)"
PROJECT_NAME="$(preview_project_name)"
HTTP_PORT="8088"
HTTPS_PORT="8443"
HTTP_PORT_EXPLICIT=0
HTTPS_PORT_EXPLICIT=0
PUBLIC_ORIGIN=""
CERT_PATH=""
KEY_PATH=""

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

LOG_ROOT="$(preview_log_root "$ROOT")"
PREVIEW_UP_LOG="$(log_file_path "$LOG_ROOT" "compose-up.log")"
PREVIEW_PS_LOG="$(log_file_path "$LOG_ROOT" "compose.ps.log")"
PREVIEW_STARTUP_LOG="$(log_file_path "$LOG_ROOT" "compose.startup.log")"
PREVIEW_UP_META_LOG="$(log_file_path "$LOG_ROOT" "up.meta.log")"
PUBLIC_ORIGIN="${HOTSPOT_PREVIEW_PUBLIC_ORIGIN:-https://localhost:${HTTPS_PORT}}"

IFS=$'\t' read -r CERT_PATH KEY_PATH < <(
  resolve_preview_tls_paths \
    "$ROOT" \
    "$PRIMARY_ROOT" \
    "${HOTSPOT_TLS_CERT_PATH:-}" \
    "${HOTSPOT_TLS_KEY_PATH:-}"
)

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

export_preview_image_defaults
export HOTSPOT_PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
export HOTSPOT_PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
export HOTSPOT_PREVIEW_HTTP_PORT="$HTTP_PORT"
export HOTSPOT_PREVIEW_HTTPS_PORT="$HTTPS_PORT"
export HOTSPOT_PREVIEW_PUBLIC_ORIGIN="$PUBLIC_ORIGIN"
export HOTSPOT_TLS_CERT_PATH="$CERT_PATH"
export HOTSPOT_TLS_KEY_PATH="$KEY_PATH"

prepare_deploy_dirs "$ROOT"
ensure_dir "$LOG_ROOT"
{
  echo "Compose project: $PROJECT_NAME"
  echo "HTTPS entrypoint: $PUBLIC_ORIGIN"
  echo "HTTP redirect entrypoint: http://localhost:${HTTP_PORT}"
  echo "TLS cert: $CERT_PATH"
  echo "TLS key: $KEY_PATH"
} | tee "$PREVIEW_UP_META_LOG"

cd "$ROOT"
run_and_capture "$PREVIEW_UP_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d
capture_compose_ps "$PROJECT_NAME" "$COMPOSE_FILE" "$PREVIEW_PS_LOG"
capture_compose_logs "$PROJECT_NAME" "$COMPOSE_FILE" "$PREVIEW_STARTUP_LOG" --tail 300

echo "Preview environment is running."
echo "Compose project: $PROJECT_NAME"
echo "HTTPS entrypoint: $PUBLIC_ORIGIN"
echo "HTTP redirect entrypoint: http://localhost:${HTTP_PORT}"
echo "Logs: $LOG_ROOT"
