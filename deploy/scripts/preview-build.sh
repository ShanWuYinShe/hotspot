#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
LOCK_PATH=""
SOURCE_ROOT="$(source_root_from_current_worktree "$ROOT")"
RELEASE_WORKTREE=""
ARTIFACT_ROOT="$(preview_artifact_root "$ROOT")"
LOG_ROOT="$(preview_log_root "$ROOT")"
BACKEND_ARTIFACT_DIR="$(backend_go_artifact_dir "$ARTIFACT_ROOT")"
FRONTEND_ARTIFACT_DIR="$(frontend_artifact_dir "$ARTIFACT_ROOT")"
BACKEND_RUNNER_IMAGE="$(backend_go_runner_image "$ROOT")"
FRONTEND_RUNNER_IMAGE="$(frontend_runner_image "$ROOT")"
PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
PREVIEW_BUILD_CONSOLE_LOG="$(log_file_path "$LOG_ROOT" "preview-build.console.log")"
BACKEND_BUILD_LOG="$(log_file_path "$LOG_ROOT" "backend-build.log")"
FRONTEND_BUILD_LOG="$(log_file_path "$LOG_ROOT" "frontend-build.log")"
BACKEND_IMAGE_BUILD_LOG="$(log_file_path "$LOG_ROOT" "backend-image-build.log")"
FRONTEND_IMAGE_BUILD_LOG="$(log_file_path "$LOG_ROOT" "frontend-image-build.log")"
PREVIEW_BUILD_META_LOG="$(log_file_path "$LOG_ROOT" "build.meta.log")"
SOURCE_REF=""

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/preview-build.sh [options]

Options:
  --ref REF           Build preview from a git ref instead of the current working tree
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --ref" >&2; exit 1; }
      SOURCE_REF="$1"
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

ensure_current_tree_flow_allowed "$ROOT" "preview-build"
prepare_deploy_dirs "$ROOT"
cleanup_path "$LOG_ROOT"
ensure_dir "$LOG_ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" preview-build)"
cleanup() {
  cleanup_path "$ARTIFACT_ROOT"
  if [[ -n "$RELEASE_WORKTREE" ]]; then
    remove_release_worktree "$ROOT" "$RELEASE_WORKTREE"
  fi
  release_lock "$LOCK_PATH"
}
trap cleanup EXIT

if [[ -n "$SOURCE_REF" ]]; then
  RELEASE_WORKTREE="$(ensure_release_worktree "$ROOT" "$SOURCE_REF" "preview-$SOURCE_REF")"
  SOURCE_ROOT="$RELEASE_WORKTREE"
fi

ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
cleanup_path "$ARTIFACT_ROOT"
ensure_dir "$BACKEND_ARTIFACT_DIR"
ensure_dir "$FRONTEND_ARTIFACT_DIR"
{
  echo "Source root: $SOURCE_ROOT"
  echo "Backend runner image: $BACKEND_RUNNER_IMAGE"
  echo "Frontend runner image: $FRONTEND_RUNNER_IMAGE"
  echo "Preview backend image: $PREVIEW_BACKEND_IMAGE"
  echo "Preview frontend image: $PREVIEW_FRONTEND_IMAGE"
  if [[ -n "$SOURCE_REF" ]]; then
    echo "Source ref: $SOURCE_REF"
  fi
} | tee "$PREVIEW_BUILD_META_LOG"

run_and_capture "$BACKEND_BUILD_LOG" docker run --rm \
  --user "$(host_uid):$(host_gid)" \
  -e HOME="$(runner_home_dir)" \
  -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
  -v "$SOURCE_ROOT:/workspace:ro" \
  -v "$BACKEND_ARTIFACT_DIR:/out" \
  "$BACKEND_RUNNER_IMAGE" \
  sh -c 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/backend-go && cp -a /workspace/backend-go/. /tmp/workspace/backend-go/ && cd /tmp/workspace/backend-go && go build -o /out/backend-go ./cmd/server'

run_and_capture "$FRONTEND_BUILD_LOG" docker run --rm \
  --user "$(host_uid):$(host_gid)" \
  -e HOME="$(runner_home_dir)" \
  -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
  -v "$SOURCE_ROOT:/workspace:ro" \
  -v "$FRONTEND_ARTIFACT_DIR:/out" \
  "$FRONTEND_RUNNER_IMAGE" \
  sh -c 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/frontend_svelte && cp -a /workspace/frontend_svelte/. /tmp/workspace/frontend_svelte/ && cd /tmp/workspace/frontend_svelte && CI=true pnpm install --frozen-lockfile && CI=true pnpm build && cp package.json pnpm-lock.yaml /out/ && rm -rf /out/build && cp -R build /out/build'

run_and_capture "$BACKEND_IMAGE_BUILD_LOG" build_app_image "$PREVIEW_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.preview.Dockerfile" "$BACKEND_ARTIFACT_DIR"
run_and_capture "$FRONTEND_IMAGE_BUILD_LOG" build_app_image "$PREVIEW_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.preview.Dockerfile" "$FRONTEND_ARTIFACT_DIR"

echo "Preview build completed." | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
echo "Backend image: $PREVIEW_BACKEND_IMAGE" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
echo "Frontend image: $PREVIEW_FRONTEND_IMAGE" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
if [[ -n "$SOURCE_REF" ]]; then
  echo "Source ref: $SOURCE_REF" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
  echo "Source root: $SOURCE_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
else
  echo "Source root: $SOURCE_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
fi
echo "Logs: $LOG_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
