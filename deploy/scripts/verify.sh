#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
COMPOSE_FILE="$ROOT/deploy/compose/verify.yml"
ENV_NAME="$(detect_verify_env_name "$ROOT")"
PROJECT_NAME=""
LOCK_PATH=""
SOURCE_ROOT="$(source_root_from_current_worktree "$ROOT")"
BACKEND_RUNNER_IMAGE="$(backend_go_runner_image "$ROOT")"
FRONTEND_RUNNER_IMAGE="$(frontend_runner_image "$ROOT")"

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/verify.sh [options]

Options:
  --env-name NAME    Isolated environment name for this verify run
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
RUN_ID="$(timestamp_run_id)"
LOG_ROOT="$(verify_log_root "$ROOT" "$ENV_NAME" "$RUN_ID")"
VERIFY_CONSOLE_LOG="$(log_file_path "$LOG_ROOT" "verify.console.log")"
BACKEND_VERIFY_LOG="$(log_file_path "$LOG_ROOT" "backend-go-verify.log")"
FRONTEND_VERIFY_LOG="$(log_file_path "$LOG_ROOT" "frontend-verify.log")"
COMPOSE_PS_LOG="$(log_file_path "$LOG_ROOT" "compose.ps.log")"
COMPOSE_FULL_LOG="$(log_file_path "$LOG_ROOT" "compose.full.log")"
RUN_SUMMARY_LOG="$(log_file_path "$LOG_ROOT" "run.meta.log")"
ensure_current_tree_flow_allowed "$ROOT" "verify"
prepare_deploy_dirs "$ROOT"
ensure_dir "$LOG_ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" "verify-$ENV_NAME")"

cleanup() {
  local status=$?
  capture_compose_ps "$PROJECT_NAME" "$COMPOSE_FILE" "$COMPOSE_PS_LOG"
  if [[ $status -ne 0 ]]; then
    capture_compose_logs "$PROJECT_NAME" "$COMPOSE_FILE" "$COMPOSE_FULL_LOG"
  fi
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
  release_lock "$LOCK_PATH"
}
trap cleanup EXIT

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"

export HOTSPOT_SOURCE_ROOT="$SOURCE_ROOT"
export HOTSPOT_BACKEND_GO_RUNNER_IMAGE="$BACKEND_RUNNER_IMAGE"
export HOTSPOT_FRONTEND_RUNNER_IMAGE="$FRONTEND_RUNNER_IMAGE"
export HOTSPOT_DOCKER_UID="$(host_uid)"
export HOTSPOT_DOCKER_GID="$(host_gid)"
export HOTSPOT_RUNNER_HOME="$(runner_home_dir)"

cd "$ROOT"
{
  echo "Run ID: $RUN_ID"
  echo "Environment name: $ENV_NAME"
  echo "Compose project: $PROJECT_NAME"
  echo "Source root: $SOURCE_ROOT"
  echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE"
  echo "Log root: $LOG_ROOT"
} | tee "$RUN_SUMMARY_LOG"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d postgres-test redis-test 2>&1 | tee -a "$VERIFY_CONSOLE_LOG"
run_and_capture "$BACKEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm backend-go-verify
run_and_capture "$FRONTEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm frontend-verify

echo "Verify passed." | tee -a "$VERIFY_CONSOLE_LOG"
echo "Environment name: $ENV_NAME" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Compose project: $PROJECT_NAME" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Source root: $SOURCE_ROOT" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$VERIFY_CONSOLE_LOG"
