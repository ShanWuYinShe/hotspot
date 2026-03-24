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

run_and_capture "$BACKEND_IMAGE_BUILD_LOG" build_app_image "$RELEASE_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.release.Dockerfile" "$BACKEND_ARTIFACT_DIR"
run_and_capture "$FRONTEND_IMAGE_BUILD_LOG" build_app_image "$RELEASE_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.release.Dockerfile" "$FRONTEND_ARTIFACT_DIR"

cat > "$MANIFEST_PATH" <<EOF
{
  "ref": "$REF",
  "resolved_sha": "$SHA",
  "label": "$LABEL",
  "backend_image": "$RELEASE_BACKEND_IMAGE",
  "frontend_image": "$RELEASE_FRONTEND_IMAGE"
}
EOF

echo "Release build completed." | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Resolved SHA: $SHA" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Release label: $LABEL" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Backend image: $RELEASE_BACKEND_IMAGE" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Frontend image: $RELEASE_FRONTEND_IMAGE" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Manifest: $MANIFEST_PATH" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
echo "Logs: $LOG_ROOT" | tee -a "$RELEASE_BUILD_CONSOLE_LOG"
