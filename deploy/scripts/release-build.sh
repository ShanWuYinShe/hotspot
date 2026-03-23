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
BACKEND_ARTIFACT_DIR="$(backend_go_artifact_dir "$ARTIFACT_ROOT")"
FRONTEND_ARTIFACT_DIR="$(frontend_artifact_dir "$ARTIFACT_ROOT")"
RELEASE_BACKEND_IMAGE="$(release_backend_image "$LABEL")"
RELEASE_FRONTEND_IMAGE="$(release_frontend_image "$LABEL")"
MANIFEST_DIR="$(deploy_tmp_root "$ROOT")/release/$LABEL"
MANIFEST_PATH="$MANIFEST_DIR/manifest.json"

prepare_deploy_dirs "$ROOT"
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

build_app_image "$RELEASE_BACKEND_IMAGE" "$ROOT/deploy/dockerfiles/backend-go.release.Dockerfile" "$BACKEND_ARTIFACT_DIR"
build_app_image "$RELEASE_FRONTEND_IMAGE" "$ROOT/deploy/dockerfiles/frontend.release.Dockerfile" "$FRONTEND_ARTIFACT_DIR"

cat > "$MANIFEST_PATH" <<EOF
{
  "ref": "$REF",
  "resolved_sha": "$SHA",
  "label": "$LABEL",
  "backend_image": "$RELEASE_BACKEND_IMAGE",
  "frontend_image": "$RELEASE_FRONTEND_IMAGE"
}
EOF

echo "Release build completed."
echo "Resolved SHA: $SHA"
echo "Release label: $LABEL"
echo "Backend image: $RELEASE_BACKEND_IMAGE"
echo "Frontend image: $RELEASE_FRONTEND_IMAGE"
echo "Manifest: $MANIFEST_PATH"
