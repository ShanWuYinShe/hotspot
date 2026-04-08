#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/verify.yml"
ENV_NAME="$(detect_verify_env_name "$ROOT")"
PROJECT_NAME=""
REMOVE_VOLUMES=0

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/verify-inspect-down.sh [options]

Stop and optionally clean up a running verify-inspect environment.

Options:
  --env-name NAME    Environment name to stop (default: auto-detected)
  --remove           Remove containers and volumes
  -h, --help         Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-name)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --env-name" >&2; exit 1; }
      ENV_NAME="$1"
      ;;
    --remove)
      REMOVE_VOLUMES=1
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
PROJECT_NAME="$(verify_project_name "$ENV_NAME")"

if [[ $REMOVE_VOLUMES -eq 1 ]]; then
  echo "Stopping and removing inspect environment: $ENV_NAME (including volumes)"
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans
else
  echo "Stopping inspect environment: $ENV_NAME"
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down --remove-orphans
fi

echo "Done."
