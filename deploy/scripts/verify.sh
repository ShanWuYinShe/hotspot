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
ensure_current_tree_flow_allowed "$ROOT" "verify"
prepare_deploy_dirs "$ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" "verify-$ENV_NAME")"

cleanup() {
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
export HOTSPOT_GO_MOD_CACHE="$(go_mod_cache_dir "$ROOT")"
export HOTSPOT_GO_BUILD_CACHE="$(go_build_cache_dir "$ROOT")"
export HOTSPOT_PNPM_STORE_CACHE="$(pnpm_store_dir "$ROOT")"
export HOTSPOT_DOCKER_UID="$(host_uid)"
export HOTSPOT_DOCKER_GID="$(host_gid)"
export HOTSPOT_RUNNER_HOME="$(runner_home_dir)"

cd "$ROOT"
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" up -d postgres-test redis-test
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm backend-go-verify
docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" run --rm frontend-verify

echo "Verify passed."
echo "Environment name: $ENV_NAME"
echo "Compose project: $PROJECT_NAME"
echo "Source root: $SOURCE_ROOT"
echo "Runner images: $BACKEND_RUNNER_IMAGE , $FRONTEND_RUNNER_IMAGE"
