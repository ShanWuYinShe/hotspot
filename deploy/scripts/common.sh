#!/usr/bin/env bash

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

project_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd
}

is_linked_worktree() {
  local root="$1"
  [[ -f "$root/.git" ]]
}

current_worktree_label() {
  local root="$1"
  basename "$root"
}

detect_verify_env_name() {
  local root="$1"
  local value="${HOTSPOT_VERIFY_ENV_NAME:-${HOTSPOT_ENV_NAME:-}}"
  if [[ -n "$value" ]]; then
    sanitize_name "$value"
    return
  fi

  if is_linked_worktree "$root"; then
    value="$(current_worktree_label "$root")"
  elif command -v git >/dev/null 2>&1; then
    value="$(git -C "$root" branch --show-current 2>/dev/null || true)"
  fi

  if [[ -z "$value" ]]; then
    value="$(current_worktree_label "$root")"
  fi

  sanitize_name "$value"
}

verify_project_name() {
  printf 'hotspot-verify-%s\n' "$(sanitize_name "$1")"
}

preview_project_name() {
  printf 'hotspot-preview\n'
}

preview_default_ports() {
  printf '8088 8443\n'
}

port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$port" -sTCP:LISTEN -n -P >/dev/null 2>&1
    return $?
  fi
  python3 - "$port" <<'PY' >/dev/null 2>&1
import socket, sys
port = int(sys.argv[1])
sock = socket.socket()
try:
    sock.bind(("127.0.0.1", port))
except OSError:
    sys.exit(0)
else:
    sys.exit(1)
finally:
    sock.close()
PY
}

next_free_port() {
  local port="$1"
  while port_in_use "$port"; do
    port=$((port + 1))
  done
  printf '%s\n' "$port"
}

deploy_tmp_root() {
  local root="$1"
  printf '%s/tmp/deploy\n' "$root"
}

ensure_dir() {
  mkdir -p "$1"
}

prepare_deploy_dirs() {
  local root="$1"
  local deploy_tmp
  deploy_tmp="$(deploy_tmp_root "$root")"
  ensure_dir "$deploy_tmp"
  ensure_dir "$deploy_tmp/artifacts/preview"
  ensure_dir "$deploy_tmp/artifacts/release"
  ensure_dir "$deploy_tmp/cache/go/pkg"
  ensure_dir "$deploy_tmp/cache/go/build"
  ensure_dir "$deploy_tmp/cache/pnpm/store"
  ensure_dir "$deploy_tmp/locks"
  ensure_dir "$deploy_tmp/release"
  ensure_dir "$deploy_tmp/worktrees"
}

preview_artifact_root() {
  local root="$1"
  printf '%s/artifacts/preview/current\n' "$(deploy_tmp_root "$root")"
}

release_artifact_root() {
  local root="$1"
  local label="$2"
  printf '%s/artifacts/release/%s\n' "$(deploy_tmp_root "$root")" "$(sanitize_name "$label")"
}

backend_go_artifact_dir() {
  local artifact_root="$1"
  printf '%s/backend-go\n' "$artifact_root"
}

frontend_artifact_dir() {
  local artifact_root="$1"
  printf '%s/frontend\n' "$artifact_root"
}

go_mod_cache_dir() {
  local root="$1"
  printf '%s/cache/go/pkg\n' "$(deploy_tmp_root "$root")"
}

go_build_cache_dir() {
  local root="$1"
  printf '%s/cache/go/build\n' "$(deploy_tmp_root "$root")"
}

pnpm_store_dir() {
  local root="$1"
  printf '%s/cache/pnpm/store\n' "$(deploy_tmp_root "$root")"
}

lock_path() {
  local root="$1"
  local name="$2"
  printf '%s/locks/%s.lock\n' "$(deploy_tmp_root "$root")" "$name"
}

acquire_lock() {
  local root="$1"
  local name="$2"
  local path
  path="$(lock_path "$root" "$name")"
  if ! mkdir "$path" 2>/dev/null; then
    echo "another ${name} flow is already running" >&2
    exit 1
  fi
  printf '%s\n' "$path"
}

release_lock() {
  local path="$1"
  [[ -n "$path" ]] || return 0
  rmdir "$path" >/dev/null 2>&1 || true
}

cleanup_path() {
  local path="$1"
  rm -rf "$path" >/dev/null 2>&1 || true
}

source_root_from_current_worktree() {
  local root="$1"
  printf '%s\n' "$root"
}

release_worktree_dir() {
  local root="$1"
  local label="$2"
  printf '%s/worktrees/%s\n' "$(deploy_tmp_root "$root")" "$(sanitize_name "$label")"
}

ensure_release_worktree() {
  local root="$1"
  local ref="$2"
  local label="$3"
  local destination
  destination="$(release_worktree_dir "$root" "$label")"
  cleanup_path "$destination"
  git -C "$root" worktree add --detach "$destination" "$ref" >/dev/null
  printf '%s\n' "$destination"
}

remove_release_worktree() {
  local root="$1"
  local path="$2"
  git -C "$root" worktree remove --force "$path" >/dev/null 2>&1 || cleanup_path "$path"
}

resolved_git_ref() {
  local root="$1"
  local ref="$2"
  git -C "$root" rev-parse --short "$ref"
}

file_fingerprint() {
  cksum < "$1" | awk '{print $1}'
}

backend_go_runner_image() {
  local root="$1"
  local hash
  hash="$(file_fingerprint "$root/deploy/dockerfiles/backend-go.verify.Dockerfile")"
  printf 'hotspot/backend-go-runner:%s\n' "$hash"
}

frontend_runner_image() {
  local root="$1"
  local hash
  hash="$(file_fingerprint "$root/deploy/dockerfiles/frontend.verify.Dockerfile")"
  printf 'hotspot/frontend-runner:%s\n' "$hash"
}

preview_backend_image() {
  printf 'hotspot/backend-go:preview\n'
}

preview_frontend_image() {
  printf 'hotspot/frontend-svelte:preview\n'
}

release_backend_image() {
  local label="$1"
  printf 'hotspot/backend-go:release-%s\n' "$(sanitize_name "$label")"
}

release_frontend_image() {
  local label="$1"
  printf 'hotspot/frontend-svelte:release-%s\n' "$(sanitize_name "$label")"
}

ensure_runner_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  if docker image inspect "$image" >/dev/null 2>&1; then
    return 0
  fi
  docker build -t "$image" -f "$dockerfile" "$context"
}

build_app_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  docker build -t "$image" -f "$dockerfile" "$context"
}

remove_preview_images() {
  docker image rm -f "$(preview_backend_image)" >/dev/null 2>&1 || true
  docker image rm -f "$(preview_frontend_image)" >/dev/null 2>&1 || true
}

join_by() {
  local delimiter="$1"
  shift
  local first=1
  for item in "$@"; do
    if [[ $first -eq 1 ]]; then
      printf '%s' "$item"
      first=0
    else
      printf '%s%s' "$delimiter" "$item"
    fi
  done
}

contains() {
  local needle="$1"
  shift
  for item in "$@"; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}
