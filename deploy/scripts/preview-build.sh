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

ONLY_TARGETS=()

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/preview-build.sh [options]

Options:
  --only TARGETS      Comma-separated targets to build: backend, frontend (default: both)
  --ref REF           Build preview from a git ref instead of the current working tree
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --only)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --only" >&2; exit 1; }
      IFS=',' read -ra ONLY_TARGETS <<< "$1"
      ;;
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

if [[ ${#ONLY_TARGETS[@]} -eq 0 ]]; then
  ONLY_TARGETS=("backend" "frontend")
fi

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

ensure_current_tree_flow_allowed "$ROOT" "preview-build"
prepare_deploy_dirs "$ROOT"
cleanup_path "$LOG_ROOT"
ensure_dir "$LOG_ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" preview-build)"
cleanup() {
  if should_run backend; then
    cleanup_path "$BACKEND_ARTIFACT_DIR"
  fi
  if should_run frontend; then
    cleanup_path "$FRONTEND_ARTIFACT_DIR"
  fi
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

if should_run backend; then
  ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
fi
if should_run frontend; then
  ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
fi
cleanup_path "$ARTIFACT_ROOT"
if should_run backend; then
  ensure_dir "$BACKEND_ARTIFACT_DIR"
fi
if should_run frontend; then
  ensure_dir "$FRONTEND_ARTIFACT_DIR"
fi
{
  echo "Source root: $SOURCE_ROOT"
  echo "Targets: ${ONLY_TARGETS[*]}"
  echo "Backend runner image: $BACKEND_RUNNER_IMAGE"
  echo "Frontend runner image: $FRONTEND_RUNNER_IMAGE"
  echo "Preview backend image: $PREVIEW_BACKEND_IMAGE"
  echo "Preview frontend image: $PREVIEW_FRONTEND_IMAGE"
  if [[ -n "$SOURCE_REF" ]]; then
    echo "Source ref: $SOURCE_REF"
  fi
} | tee "$PREVIEW_BUILD_META_LOG"

# Artifact builds
if should_run backend && should_run frontend; then
  run_backend_artifact_build "$BACKEND_BUILD_LOG" "$SOURCE_ROOT" "$BACKEND_ARTIFACT_DIR" "$BACKEND_RUNNER_IMAGE &
  BACKEND_PID=$!
  run_frontend_artifact_build "$FRONTEND_BUILD_LOG" "$SOURCE_ROOT" "$FRONTEND_ARTIFACT_DIR" "$FRONTEND_RUNNER_IMAGE &
  FRONTEND_PID=$!
  BACKEND_EXIT=0
  FRONTEND_EXIT=0
  wait "$BACKEND_PID" || BACKEND_EXIT=$?
  wait "$FRONTEND_PID" || FRONTEND_EXIT=$?
  if [[ $BACKEND_EXIT -ne 0 || $FRONTEND_EXIT -ne 0 ]]; then
    echo "Artifact build failed." >&2
    [[ $BACKEND_EXIT -ne 0 ]] && echo "  backend artifact build exited with $BACKEND_EXIT" >&2
    [[ $FRONTEND_EXIT -ne 0 ]] && echo "  frontend artifact build exited with $FRONTEND_EXIT" >&2
    exit 1
  fi
elif should_run backend; then
  run_backend_artifact_build "$BACKEND_BUILD_LOG" "$SOURCE_ROOT" "$BACKEND_ARTIFACT_DIR" "$BACKEND_RUNNER_IMAGE"
elif should_run frontend; then
  run_frontend_artifact_build "$FRONTEND_BUILD_LOG" "$SOURCE_ROOT" "$FRONTEND_ARTIFACT_DIR" "$FRONTEND_RUNNER_IMAGE"
fi

# Image builds (sequential, depend on artifacts)
if should_run backend; then
  run_and_capture "$BACKEND_IMAGE_BUILD_LOG" build_app_image "$PREVIEW_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.app.Dockerfile" "$BACKEND_ARTIFACT_DIR" \
    "hotspot.role=preview-backend-go-image" \
    "hotspot.display.name=Backend Go Preview Image" \
    "hotspot.image.kind=preview"
fi
if should_run frontend; then
  run_and_capture "$FRONTEND_IMAGE_BUILD_LOG" build_app_image "$PREVIEW_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.app.Dockerfile" "$FRONTEND_ARTIFACT_DIR" \
    "hotspot.role=preview-frontend-image" \
    "hotspot.display.name=Frontend Preview Image" \
    "hotspot.image.kind=preview"
fi

echo "Preview build completed." | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
if should_run backend; then
  echo "Backend image: $PREVIEW_BACKEND_IMAGE" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
fi
if should_run frontend; then
  echo "Frontend image: $PREVIEW_FRONTEND_IMAGE" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
fi
echo "Targets: ${ONLY_TARGETS[*]}" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
if [[ -n "$SOURCE_REF" ]]; then
  echo "Source ref: $SOURCE_REF" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
  echo "Source root: $SOURCE_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
else
  echo "Source root: $SOURCE_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
fi
echo "Logs: $LOG_ROOT" | tee -a "$PREVIEW_BUILD_CONSOLE_LOG"
