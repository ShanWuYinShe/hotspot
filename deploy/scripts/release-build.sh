#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
REF="HEAD"
LABEL=""
LOCK_PATH=""
SOURCE_ROOT=""
RELEASE_WORKTREE=""
ARTIFACT_ROOT=""
LOG_ROOT=""
BACKEND_RUNNER_IMAGE="$(backend_go_runner_image "$ROOT")"
FRONTEND_RUNNER_IMAGE="$(frontend_runner_image "$ROOT")"

usage() {
  cat <<'EOF'
Usage: ./deploy/scripts/release-build.sh [options]

Options:
  --ref REF           Git ref to build from (default: HEAD)
  --label LABEL       Optional release label; defaults to resolved short SHA
  -h, --help          Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --ref" >&2; exit 1; }
      REF="$1"
      ;;
    --label)
      shift
      [[ $# -gt 0 ]] || { echo "missing value for --label" >&2; exit 1; }
      LABEL="$1"
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

ensure_current_tree_flow_allowed "$ROOT" "release-build"
SHA="$(resolved_git_ref "$ROOT" "$REF")"
if [[ -z "$LABEL" ]]; then
  LABEL="$SHA"
fi
LABEL="$(sanitize_name "$LABEL")"
ARTIFACT_ROOT="$(release_artifact_root "$ROOT" "$LABEL")"
LOG_ROOT="$(release_log_root "$ROOT" "$LABEL")"
BACKEND_ARTIFACT_DIR="$(backend_go_artifact_dir "$ARTIFACT_ROOT")"
FRONTEND_ARTIFACT_DIR="$(frontend_artifact_dir "$ARTIFACT_ROOT")"
RELEASE_BACKEND_IMAGE="$(release_backend_image "$LABEL")"
RELEASE_FRONTEND_IMAGE="$(release_frontend_image "$LABEL")"
MANIFEST_DIR="$(deploy_tmp_root "$ROOT")/release/$LABEL"
MANIFEST_PATH="$MANIFEST_DIR/manifest.json"
RELEASE_BUILD_CONSOLE_LOG="$(log_file_path "$LOG_ROOT" "release-build.console.log")"
BACKEND_BUILD_LOG="$(log_file_path "$LOG_ROOT" "backend-build.log")"
FRONTEND_BUILD_LOG="$(log_file_path "$LOG_ROOT" "frontend-build.log")"
BACKEND_IMAGE_BUILD_LOG="$(log_file_path "$LOG_ROOT" "backend-image-build.log")"
FRONTEND_IMAGE_BUILD_LOG="$(log_file_path "$LOG_ROOT" "frontend-image-build.log")"
RELEASE_BUILD_META_LOG="$(log_file_path "$LOG_ROOT" "release.meta.log")"

prepare_deploy_dirs "$ROOT"
cleanup_path "$LOG_ROOT"
ensure_dir "$LOG_ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" "release-$LABEL")"
cleanup() {
  if [[ -n "$RELEASE_WORKTREE" ]]; then
    remove_release_worktree "$ROOT" "$RELEASE_WORKTREE"
  fi
  release_lock "$LOCK_PATH"
}
trap cleanup EXIT

RELEASE_WORKTREE="$(ensure_release_worktree "$ROOT" "$REF" "$LABEL")"
SOURCE_ROOT="$RELEASE_WORKTREE"
ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
cleanup_path "$ARTIFACT_ROOT"
ensure_dir "$BACKEND_ARTIFACT_DIR"
ensure_dir "$FRONTEND_ARTIFACT_DIR"
ensure_dir "$MANIFEST_DIR"
{
  echo "Ref: $REF"
  echo "Resolved SHA: $SHA"
  echo "Label: $LABEL"
  echo "Source root: $SOURCE_ROOT"
  echo "Backend runner image: $BACKEND_RUNNER_IMAGE"
  echo "Frontend runner image: $FRONTEND_RUNNER_IMAGE"
  echo "Backend release image: $RELEASE_BACKEND_IMAGE"
  echo "Frontend release image: $RELEASE_FRONTEND_IMAGE"
} | tee "$RELEASE_BUILD_META_LOG"

run_backend_artifact_build "$BACKEND_BUILD_LOG" "$SOURCE_ROOT" "$BACKEND_ARTIFACT_DIR" "$BACKEND_RUNNER_IMAGE" &
BACKEND_PID=$!
run_frontend_artifact_build "$FRONTEND_BUILD_LOG" "$SOURCE_ROOT" "$FRONTEND_ARTIFACT_DIR" "$FRONTEND_RUNNER_IMAGE" &
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

run_and_capture "$BACKEND_IMAGE_BUILD_LOG" build_app_image "$RELEASE_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.app.Dockerfile" "$BACKEND_ARTIFACT_DIR" \
  "hotspot.role=release-backend-go-image" \
  "hotspot.display.name=Backend Go Release Image" \
  "hotspot.image.kind=release" \
  "hotspot.release.label=$LABEL" \
  "hotspot.release.sha=$SHA"
run_and_capture "$FRONTEND_IMAGE_BUILD_LOG" build_app_image "$RELEASE_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.app.Dockerfile" "$FRONTEND_ARTIFACT_DIR" \
  "hotspot.role=release-frontend-image" \
  "hotspot.display.name=Frontend Release Image" \
  "hotspot.image.kind=release" \
  "hotspot.release.label=$LABEL" \
  "hotspot.release.sha=$SHA"

cat > "$MANIFEST_PATH" <<EOF
{
  "ref": "$REF",
  "resolved_sha": "$SHA",
  "label": "$LABEL",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "source_root": "$SOURCE_ROOT",
  "backend_runner_image": "$BACKEND_RUNNER_IMAGE",
  "frontend_runner_image": "$FRONTEND_RUNNER_IMAGE",
  "base_images": {
    "golang": "${HOTSPOT_GOLANG_BASE_IMAGE:-$(default_golang_base_image)}",
    "node": "${HOTSPOT_NODE_BASE_IMAGE:-$(default_node_base_image)}",
    "alpine": "${HOTSPOT_ALPINE_BASE_IMAGE:-$(default_alpine_base_image)}"
  },
  "backend_image": "$RELEASE_BACKEND_IMAGE",
  "frontend_image": "$RELEASE_FRONTEND_IMAGE",
  "log_root": "$LOG_ROOT"
}
EOF

echo "Release build completed." | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Resolved SHA: $SHA" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Release label: $LABEL" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Backend image: $RELEASE_BACKEND_IMAGE" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Frontend image: $RELEASE_FRONTEND_IMAGE" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Manifest: $MANIFEST_PATH" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
