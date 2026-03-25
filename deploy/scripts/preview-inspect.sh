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

export_preview_image_defaults
export HOTSPOT_PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
export HOTSPOT_PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
export HOTSPOT_TLS_CERT_PATH="${HOTSPOT_TLS_CERT_PATH:-/tmp/missing.crt}"
export HOTSPOT_TLS_KEY_PATH="${HOTSPOT_TLS_KEY_PATH:-/tmp/missing.key}"

LOG_ROOT="$(preview_log_root "$ROOT")"
INSPECT_CONSOLE_LOG="$(log_file_path "$LOG_ROOT" "inspect.console.log")"
INSPECT_PS_LOG="$(log_file_path "$LOG_ROOT" "compose.ps.inspect.log")"
INSPECT_FULL_LOG="$(log_file_path "$LOG_ROOT" "compose.full.inspect.log")"
INSPECT_HOME_LOG="$(log_file_path "$LOG_ROOT" "curl-home.log")"
INSPECT_HEALTH_LOG="$(log_file_path "$LOG_ROOT" "curl-health.log")"
INSPECT_S3_LOG="$(log_file_path "$LOG_ROOT" "curl-s3.log")"
INSPECT_META_LOG="$(log_file_path "$LOG_ROOT" "inspect.meta.log")"
BASE_URL="https://${HOSTNAME}:${HTTPS_PORT}"
prepare_deploy_dirs "$ROOT"
ensure_dir "$LOG_ROOT"
cd "$ROOT"

{
  echo "Compose project: $PROJECT_NAME"
  echo "Backend image: $HOTSPOT_PREVIEW_BACKEND_IMAGE"
  echo "Frontend image: $HOTSPOT_PREVIEW_FRONTEND_IMAGE"
  echo "Base URL: $BASE_URL"
  echo "Tail lines: $TAIL_LINES"
} | tee "$INSPECT_META_LOG"

echo "Compose project: $PROJECT_NAME" | tee "$INSPECT_CONSOLE_LOG"
echo "Backend image: $HOTSPOT_PREVIEW_BACKEND_IMAGE" | tee -a "$INSPECT_CONSOLE_LOG"
echo "Frontend image: $HOTSPOT_PREVIEW_FRONTEND_IMAGE" | tee -a "$INSPECT_CONSOLE_LOG"
echo | tee -a "$INSPECT_CONSOLE_LOG"

echo "[preview] status" | tee -a "$INSPECT_CONSOLE_LOG"
capture_compose_ps "$PROJECT_NAME" "$COMPOSE_FILE" "$INSPECT_PS_LOG"
cat "$INSPECT_PS_LOG"

run_logs() {
  echo | tee -a "$INSPECT_CONSOLE_LOG"
  echo "[preview] logs" | tee -a "$INSPECT_CONSOLE_LOG"
  capture_compose_logs "$PROJECT_NAME" "$COMPOSE_FILE" "$INSPECT_FULL_LOG"
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" logs --tail "$TAIL_LINES" 2>&1 | tee -a "$INSPECT_CONSOLE_LOG"
}

echo | tee -a "$INSPECT_CONSOLE_LOG"
echo "[preview] checks" | tee -a "$INSPECT_CONSOLE_LOG"
if [[ $SKIP_HOME -eq 0 ]]; then
  echo "GET /" | tee -a "$INSPECT_CONSOLE_LOG"
  curl -k -f -sS "$BASE_URL/" 2>&1 | tee "$INSPECT_HOME_LOG" >/dev/null
  echo "ok" | tee -a "$INSPECT_CONSOLE_LOG"
fi

if [[ $SKIP_HEALTH -eq 0 ]]; then
  echo | tee -a "$INSPECT_CONSOLE_LOG"
  echo "POST /api/v1/system/health" | tee -a "$INSPECT_CONSOLE_LOG"
  curl -k -f -sS -X POST "$BASE_URL/api/v1/system/health" -H "Content-Type: application/json" -d "{}" 2>&1 | tee "$INSPECT_HEALTH_LOG"
  echo | tee -a "$INSPECT_CONSOLE_LOG"
fi

if [[ $SKIP_S3 -eq 0 ]]; then
  echo | tee -a "$INSPECT_CONSOLE_LOG"
  echo "GET /s3/" | tee -a "$INSPECT_CONSOLE_LOG"
  s3_body="$(log_file_path "$LOG_ROOT" "curl-s3.body.log")"
  s3_code_file="$(log_file_path "$LOG_ROOT" "curl-s3.code.log")"
  curl -k -sS -o "$s3_body" -w "%{http_code}" "$BASE_URL/s3/" >"$s3_code_file"
  code="$(cat "$s3_code_file")"
  {
    echo "status=$code"
    cat "$s3_body"
  } >"$INSPECT_S3_LOG"
  if [[ "$code" == "200" || "$code" == "403" || "$code" == "404" ]]; then
    echo "ok (status $code)" | tee -a "$INSPECT_CONSOLE_LOG"
  else
    echo "unexpected status $code" | tee -a "$INSPECT_CONSOLE_LOG" >&2
    run_logs
    exit 1
  fi
fi

if [[ $FORCE_LOGS -eq 1 ]]; then
  run_logs
fi

echo | tee -a "$INSPECT_CONSOLE_LOG"
echo "Preview inspection passed." | tee -a "$INSPECT_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$INSPECT_CONSOLE_LOG"
