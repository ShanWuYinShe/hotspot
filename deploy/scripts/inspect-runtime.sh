#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$ROOT/common.sh"
REPO_ROOT="$(cd "$ROOT/../.." && pwd)"
ENV_NAME="$(detect_env_name "$REPO_ROOT")"
read -r _ DEFAULT_HTTPS_PORT <<< "$(derive_runtime_ports "$ENV_NAME")"
HTTPS_PORT="$DEFAULT_HTTPS_PORT"
HOSTNAME="localhost"
LOG_SERVICES="backend,frontend,nginx"
LOG_TAIL="300"
SKIP_STATUS=0
SKIP_CHECK=0
FORCE_LOGS=0

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/inspect-runtime.sh [options]

Options:
  --env-name NAME       Target isolated environment name (default: default)
  --https-port PORT     HTTPS port to check (default: 8443)
  --host HOST           Hostname to check (default: localhost)
  --log-services a,b,c  Services to show when logs are needed (default: backend,frontend,nginx)
  --log-tail N          Number of log lines to show (default: 300)
  --skip-status         Skip status-runtime step
  --skip-check          Skip check-runtime step
  --logs                Always show logs, even if checks pass
  -h, --help            Show this help message
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
    --log-services)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --log-services" >&2; exit 1; }
      LOG_SERVICES="$1"
      ;;
    --log-tail)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --log-tail" >&2; exit 1; }
      LOG_TAIL="$1"
      ;;
    --skip-status)
      SKIP_STATUS=1
      ;;
    --skip-check)
      SKIP_CHECK=1
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

run_logs() {
  echo
  echo "[inspect] logs"
  bash "$ROOT/logs-runtime.sh" --env-name "$ENV_NAME" --services "$LOG_SERVICES" --tail "$LOG_TAIL"
}

if [[ $SKIP_STATUS -eq 0 ]]; then
  echo "[inspect] status"
  bash "$ROOT/status-runtime.sh" --env-name "$ENV_NAME"
fi

if [[ $SKIP_CHECK -eq 0 ]]; then
  echo
  echo "[inspect] checks"
  if ! bash "$ROOT/check-runtime.sh" --env-name "$ENV_NAME" --https-port "$HTTPS_PORT" --host "$HOSTNAME"; then
    echo
    echo "Runtime inspection failed. Showing logs..." >&2
    run_logs
    exit 1
  fi
fi

if [[ $FORCE_LOGS -eq 1 ]]; then
  run_logs
fi

echo

echo "Runtime inspection passed."
