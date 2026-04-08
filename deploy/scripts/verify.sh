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

# --only targets: "backend" and/or "frontend"
ONLY_TARGETS=()

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/verify.sh [options]

verify 负责静态检查、构建与测试；若需要页面、console、关键 UI 与交互自动确认，请改用 ./deploy/scripts/verify-inspect.sh。

Options:
  --only TARGETS     Comma-separated targets to verify: backend, frontend (default: both)
  --env-name NAME    Isolated environment name for this verify run
  -h, --help         Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --only)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --only" >&2; exit 1; }
      IFS=',' read -ra ONLY_TARGETS <<< "$1"
      ;;
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

# Default to both targets
if [[ ${#ONLY_TARGETS[@]} -eq 0 ]]; then
  ONLY_TARGETS=("backend" "frontend")
fi

# Validate targets
for target in "${ONLY_TARGETS[@]}"; do
  if [[ "$target" != "backend" && "$target" != "frontend" ]]; then
    echo "invalid --only target: $target (expected: backend, frontend)" >&2
    exit 1
  fi
done

should_run() {
  local target="$1"
  local t
  for t in "${ONLY_TARGETS[@]}"; do
    [[ "$t" == "$target" ]] && return 0
  done
  return 1
}

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

# Build runner images only for selected targets
if should_run backend; then
  ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
fi
if should_run frontend; then
  ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
fi
export_verify_image_defaults

export HOTSPOT_SOURCE_ROOT="$SOURCE_ROOT"
export HOTSPOT_BACKEND_GO_RUNNER_IMAGE="$BACKEND_RUNNER_IMAGE"
export HOTSPOT_FRONTEND_RUNNER_IMAGE="$FRONTEND_RUNNER_IMAGE"
export HOTSPOT_VERIFY_NGINX_IMAGE="${HOTSPOT_VERIFY_NGINX_IMAGE:-$(default_preview_nginx_image)}"
export HOTSPOT_DOCKER_UID="$(host_uid)"
export HOTSPOT_DOCKER_GID="$(host_gid)"
export HOTSPOT_RUNNER_HOME="$(runner_home_dir)"

cd "$ROOT"
{
  echo "Run ID: $RUN_ID"
  echo "Environment name: $ENV_NAME"
  echo "Compose project: $PROJECT_NAME"
  echo "Source root: $SOURCE_ROOT"
  echo "Targets: ${ONLY_TARGETS[*]}"
  echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE"
  echo "Log root: $LOG_ROOT"
} | tee "$RUN_SUMMARY_LOG"

# Start infra only when backend is selected (frontend-verify has no depends_on)
if should_run backend; then
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d postgres redis 2>&1 | tee -a "$VERIFY_CONSOLE_LOG"
fi

# Execute verify
if should_run backend && should_run frontend; then
  # Run both in parallel
  run_and_capture "$BACKEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm backend-go-verify &
  BACKEND_PID=$!
  run_and_capture "$FRONTEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm frontend-verify &
  FRONTEND_PID=$!

  BACKEND_EXIT=0
  FRONTEND_EXIT=0
  wait "$BACKEND_PID" || BACKEND_EXIT=$?
  wait "$FRONTEND_PID" || FRONTEND_EXIT=$?

  if [[ $BACKEND_EXIT -ne 0 || $FRONTEND_EXIT -ne 0 ]]; then
    echo "Verify failed." >&2
    [[ $BACKEND_EXIT -ne 0 ]] && echo "  backend-go-verify exited with $BACKEND_EXIT" >&2
    [[ $FRONTEND_EXIT -ne 0 ]] && echo "  frontend-verify exited with $FRONTEND_EXIT" >&2
    exit 1
  fi
elif should_run backend; then
  run_and_capture "$BACKEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm backend-go-verify
elif should_run frontend; then
  run_and_capture "$FRONTEND_VERIFY_LOG" docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm frontend-verify
fi

echo "Verify passed." | tee -a "$VERIFY_CONSOLE_LOG"
echo "Environment name: $ENV_NAME" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Compose project: $PROJECT_NAME" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Source root: $SOURCE_ROOT" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Targets: ${ONLY_TARGETS[*]}" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE" | tee -a "$VERIFY_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$VERIFY_CONSOLE_LOG"
