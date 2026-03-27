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
HTTP_PORT="38088"
RUN_ID=""
LOG_ROOT=""
INSPECT_CONSOLE_LOG=""
COMPOSE_PS_LOG=""
COMPOSE_FULL_LOG=""
RUN_SUMMARY_LOG=""

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/verify-inspect.sh [options]

verify-inspect 只负责在 Docker 中拉起供 Claude 使用的隔离运行环境与入口，不负责定义或执行具体验证哪些页面/交互。

Options:
  --env-name NAME    Isolated environment name for this inspect run
  --http-port PORT   Public HTTP port for inspect nginx entry (default: 38088)
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
    --http-port)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --http-port" >&2; exit 1; }
      HTTP_PORT="$1"
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
INSPECT_CONSOLE_LOG="$(log_file_path "$LOG_ROOT" "verify-inspect.console.log")"
COMPOSE_PS_LOG="$(log_file_path "$LOG_ROOT" "compose.inspect.ps.log")"
COMPOSE_FULL_LOG="$(log_file_path "$LOG_ROOT" "compose.inspect.full.log")"
RUN_SUMMARY_LOG="$(log_file_path "$LOG_ROOT" "inspect.meta.log")"

ensure_current_tree_flow_allowed "$ROOT" "verify-inspect"
prepare_deploy_dirs "$ROOT"
ensure_dir "$LOG_ROOT"
docker compose -p "$(verify_project_name "$ENV_NAME")" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
LOCK_PATH="$(acquire_lock "$ROOT" "verify-inspect-$ENV_NAME")"

cleanup() {
  local status=$?
  capture_compose_ps "$PROJECT_NAME" "$COMPOSE_FILE" "$COMPOSE_PS_LOG"
  if [[ $status -ne 0 ]]; then
    capture_compose_logs "$PROJECT_NAME" "$COMPOSE_FILE" "$COMPOSE_FULL_LOG"
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
  fi
  release_lock "$LOCK_PATH"
}
trap cleanup EXIT

if port_in_use "$HTTP_PORT"; then
  echo "verify-inspect HTTP port $HTTP_PORT is already in use" >&2
  exit 1
fi

ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
export_verify_image_defaults

export HOTSPOT_SOURCE_ROOT="$SOURCE_ROOT"
export HOTSPOT_BACKEND_GO_RUNNER_IMAGE="$BACKEND_RUNNER_IMAGE"
export HOTSPOT_FRONTEND_RUNNER_IMAGE="$FRONTEND_RUNNER_IMAGE"
export HOTSPOT_DOCKER_UID="$(host_uid)"
export HOTSPOT_DOCKER_GID="$(host_gid)"
export HOTSPOT_RUNNER_HOME="$(runner_home_dir)"
export HOTSPOT_VERIFY_INSPECT_HTTP_PORT="$HTTP_PORT"
export HOTSPOT_VERIFY_INSPECT_BASE_URL="http://verify-inspect-nginx"
export HOTSPOT_VERIFY_INSPECT_LOG_ROOT="/inspect-logs"
export HOTSPOT_VERIFY_INSPECT_HOST_LOG_ROOT="$LOG_ROOT"
export HOTSPOT_VERIFY_NGINX_IMAGE="${HOTSPOT_VERIFY_NGINX_IMAGE:-$(default_preview_nginx_image)}"

cd "$ROOT"
{
  echo "Run ID: $RUN_ID"
  echo "Environment name: $ENV_NAME"
  echo "Compose project: $PROJECT_NAME"
  echo "Source root: $SOURCE_ROOT"
  echo "Inspect entrypoint: http://localhost:$HTTP_PORT"
  echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE"
  echo "Log root: $LOG_ROOT"
} | tee "$RUN_SUMMARY_LOG"

docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d postgres-test redis-test backend-go-inspect frontend-inspect verify-inspect-nginx 2>&1 | tee -a "$INSPECT_CONSOLE_LOG"
ready=0
for _ in $(seq 1 180); do
  if curl -f -sS "http://localhost:$HTTP_PORT/" >/dev/null 2>&1 && curl -f -sS -X POST "http://localhost:$HTTP_PORT/api/v1/system/health" -H "Content-Type: application/json" -d '{}' >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
if [[ $ready -ne 1 ]]; then
  echo "verify-inspect environment did not become ready in time" >&2
  exit 1
fi
echo "Verify inspect environment is ready." | tee -a "$INSPECT_CONSOLE_LOG"
echo "Environment name: $ENV_NAME" | tee -a "$INSPECT_CONSOLE_LOG"
echo "Compose project: $PROJECT_NAME" | tee -a "$INSPECT_CONSOLE_LOG"
echo "Inspect entrypoint: http://localhost:$HTTP_PORT" | tee -a "$INSPECT_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$INSPECT_CONSOLE_LOG"
echo "下一步应由 Claude Code 主动访问该入口做页面、console、UI 与交互检查；deploy 不承载具体页面测试用例。" | tee -a "$INSPECT_CONSOLE_LOG"
