#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
ROOT="$(project_root)"
LOCK_PATH=""
SNAPSHOT_DIR="$(preview_snapshot_dir "$ROOT")"
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

prepare_deploy_dirs "$ROOT"
LOCK_PATH="$(acquire_lock "$ROOT" preview-build)"
cleanup() {
  cleanup_path "$SNAPSHOT_DIR"
  cleanup_path "$ARTIFACT_ROOT"
  release_lock "$LOCK_PATH"
}
trap cleanup EXIT

if [[ -n "$SOURCE_REF" ]]; then
  copy_git_ref_snapshot "$ROOT" "$SOURCE_REF" "$SNAPSHOT_DIR"
else
  copy_worktree_snapshot "$ROOT" "$SNAPSHOT_DIR"
fi

ensure_runner_image "$BACKEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.verify.Dockerfile" "$ROOT"
ensure_runner_image "$FRONTEND_RUNNER_IMAGE" "$ROOT/deploy/dockerfiles/frontend.verify.Dockerfile" "$ROOT"
cleanup_path "$ARTIFACT_ROOT"
ensure_dir "$BACKEND_ARTIFACT_DIR"
ensure_dir "$FRONTEND_ARTIFACT_DIR"

docker run --rm \
  -v "$SNAPSHOT_DIR:/workspace:ro" \
  -v "$BACKEND_ARTIFACT_DIR:/out" \
  -v "$(go_mod_cache_dir "$ROOT"):/cache/go/pkg" \
  -v "$(go_build_cache_dir "$ROOT"):/cache/go/build" \
  -w /workspace/backend-go \
  "$BACKEND_RUNNER_IMAGE" \
  sh -c 'go build -o /out/backend-go ./cmd/server'

docker run --rm \
  -v "$SNAPSHOT_DIR:/workspace" \
  -v "$(pnpm_store_dir "$ROOT"):/cache/pnpm/store" \
  -w /workspace/frontend_svelte \
  "$FRONTEND_RUNNER_IMAGE" \
  sh -c 'pnpm install --frozen-lockfile && pnpm build'

cp "$SNAPSHOT_DIR/frontend_svelte/package.json" "$SNAPSHOT_DIR/frontend_svelte/pnpm-lock.yaml" "$FRONTEND_ARTIFACT_DIR/"
rm -rf "$FRONTEND_ARTIFACT_DIR/build"
cp -R "$SNAPSHOT_DIR/frontend_svelte/build" "$FRONTEND_ARTIFACT_DIR/build"

build_app_image "$PREVIEW_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.preview.Dockerfile" "$BACKEND_ARTIFACT_DIR"
build_app_image "$PREVIEW_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.preview.Dockerfile" "$FRONTEND_ARTIFACT_DIR"

echo "Preview build completed."
echo "Backend image: $PREVIEW_BACKEND_IMAGE"
echo "Frontend image: $PREVIEW_FRONTEND_IMAGE"
if [[ -n "$SOURCE_REF" ]]; then
  echo "Source ref: $SOURCE_REF"
else
  echo "Source: current working tree snapshot"
fi
