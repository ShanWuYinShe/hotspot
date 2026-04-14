#!/usr/bin/env bash

deploy_tmp_root() {
  local root="$1"
  printf '%s/.local/deploy\n' "$root"
}

ensure_dir() {
  mkdir -p "$1"
}

ensure_writable_dir() {
  local path="$1"
  if [[ ! -w "$path" ]]; then
    echo "deploy path is not writable by the current user: $path" >&2
    echo "fix the ownership of this .local/deploy path before rerunning the flow." >&2
    exit 1
  fi
}

prepare_deploy_dirs() {
  local root="$1"
  local deploy_tmp
  deploy_tmp="$(deploy_tmp_root "$root")"
  ensure_dir "$deploy_tmp"
  ensure_dir "$deploy_tmp/artifacts/preview"
  ensure_dir "$deploy_tmp/artifacts/release"
  ensure_dir "$deploy_tmp/locks"
  ensure_dir "$deploy_tmp/logs"
  ensure_dir "$deploy_tmp/logs/verify"
  ensure_dir "$deploy_tmp/logs/preview/current"
  ensure_dir "$deploy_tmp/logs/release"
  ensure_dir "$deploy_tmp/release"
  ensure_dir "$deploy_tmp/worktrees"
  ensure_writable_dir "$deploy_tmp"
  ensure_writable_dir "$deploy_tmp/artifacts/preview"
  ensure_writable_dir "$deploy_tmp/artifacts/release"
  ensure_writable_dir "$deploy_tmp/locks"
  ensure_writable_dir "$deploy_tmp/logs"
  ensure_writable_dir "$deploy_tmp/logs/verify"
  ensure_writable_dir "$deploy_tmp/logs/preview/current"
  ensure_writable_dir "$deploy_tmp/logs/release"
  ensure_writable_dir "$deploy_tmp/release"
  ensure_writable_dir "$deploy_tmp/worktrees"
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

verify_log_root() {
  local root="$1"
  local env_name="$2"
  local run_id="$3"
  printf '%s/logs/verify/%s/%s\n' "$(deploy_tmp_root "$root")" "$(sanitize_name "$env_name")" "$(sanitize_name "$run_id")"
}

preview_log_root() {
  local root="$1"
  printf '%s/logs/preview/current\n' "$(deploy_tmp_root "$root")"
}

release_log_root() {
  local root="$1"
  local label="$2"
  printf '%s/logs/release/%s\n' "$(deploy_tmp_root "$root")" "$(sanitize_name "$label")"
}

log_file_path() {
  local root="$1"
  local name="$2"
  printf '%s/%s\n' "$root" "$name"
}

backend_go_artifact_dir() {
  local artifact_root="$1"
  printf '%s/backend-go\n' "$artifact_root"
}

frontend_artifact_dir() {
  local artifact_root="$1"
  printf '%s/frontend\n' "$artifact_root"
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
  if [[ ! -e "$path" ]]; then
    return 0
  fi
  chmod -R u+rwX "$path" >/dev/null 2>&1 || true
  if ! rm -rf "$path"; then
    echo "failed to remove path: $path" >&2
    return 1
  fi
}

run_and_capture() {
  local log_path="$1"
  shift
  mkdir -p "$(dirname "$log_path")"
  "$@" 2>&1 | tee "$log_path"
}

capture_compose_ps() {
  local project_name="$1"
  local compose_file="$2"
  local log_path="$3"
  mkdir -p "$(dirname "$log_path")"
  docker compose -p "$project_name" -f "$compose_file" ps >"$log_path" 2>&1 || true
}

capture_compose_logs() {
  local project_name="$1"
  local compose_file="$2"
  local log_path="$3"
  shift 3
  mkdir -p "$(dirname "$log_path")"
  docker compose -p "$project_name" -f "$compose_file" logs --no-color "$@" >"$log_path" 2>&1 || true
}
