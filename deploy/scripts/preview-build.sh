#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
LOCK_PATH=""
SOURCE_ROOT="$(source_root_from_current_worktree "$ROOT")"
RELEASE_WORKTREE=""
ARTIFACT_ROOT="$(preview_artifact_root "$ROOT")"
BACKEND_ARTIFACT_DIR="$(backend_go_artifact_dir "$ARTIFACT_ROOT")"
FRONTEND_ARTIFACT_DIR="$(frontend_artifact_dir "$ARTIFACT_ROOT")"
BACKEND_RUNNER_IMAGE="$(backend_go_runner_image "$ROOT")"
FRONTEND_RUNNER_IMAGE="$(frontend_runner_image "$ROOT")"
PREVIEW_BACKEND_IMAGE="$(preview_backend_image)"
PREVIEW_FRONTEND_IMAGE="$(preview_frontend_image)"
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

docker run --rm \
  --user "$(host_uid):$(host_gid)" \
  -e HOME="$(runner_home_dir)" \
  -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
  -v "$SOURCE_ROOT:/workspace:ro" \
  -v "$BACKEND_ARTIFACT_DIR:/out" \
  -v "$(go_mod_cache_dir "$ROOT"):/cache/go/pkg" \
  -v "$(go_build_cache_dir "$ROOT"):/cache/go/build" \
  "$BACKEND_RUNNER_IMAGE" \
  sh -c 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/backend-go && cp -a /workspace/backend-go/. /tmp/workspace/backend-go/ && cd /tmp/workspace/backend-go && go build -o /out/backend-go ./cmd/server'

docker run --rm \
  --user "$(host_uid):$(host_gid)" \
  -e HOME="$(runner_home_dir)" \
  -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
  -v "$SOURCE_ROOT:/workspace:ro" \
  -v "$FRONTEND_ARTIFACT_DIR:/out" \
  -v "$(pnpm_store_dir "$ROOT"):/cache/pnpm/store" \
  "$FRONTEND_RUNNER_IMAGE" \
  sh -c 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/frontend_svelte && cp -a /workspace/frontend_svelte/. /tmp/workspace/frontend_svelte/ && cd /tmp/workspace/frontend_svelte && CI=true pnpm install --frozen-lockfile && CI=true pnpm build && cp package.json pnpm-lock.yaml /out/ && rm -rf /out/build && cp -R build /out/build'

build_app_image "$PREVIEW_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.preview.Dockerfile" "$BACKEND_ARTIFACT_DIR"
build_app_image "$PREVIEW_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.preview.Dockerfile" "$FRONTEND_ARTIFACT_DIR"

echo "Preview build completed."
echo "Backend image: $PREVIEW_BACKEND_IMAGE"
echo "Frontend image: $PREVIEW_FRONTEND_IMAGE"
if [[ -n "$SOURCE_REF" ]]; then
  echo "Source ref: $SOURCE_REF"
  echo "Source root: $SOURCE_ROOT"
else
  echo "Source root: $SOURCE_ROOT"
fi
