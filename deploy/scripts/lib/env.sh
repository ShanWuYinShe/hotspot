#!/usr/bin/env bash

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

project_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd
}

is_linked_worktree() {
  local root="$1"
  [[ -f "$root/.git" ]]
}

current_worktree_label() {
  local root="$1"
  basename "$root"
}

current_git_branch() {
  local root="$1"
  git -C "$root" branch --show-current 2>/dev/null || true
}

ensure_current_tree_flow_allowed() {
  local root="$1"
  local flow_name="$2"
  local branch
  local status

  if is_linked_worktree "$root"; then
    return 0
  fi

  branch="$(current_git_branch "$root")"
  if [[ "$branch" != "main" ]]; then
    echo "$flow_name must run from a linked worktree when the primary checkout is not on main (current: ${branch:-detached})." >&2
    echo "Keep the primary checkout on main and move actual changes into a worktree." >&2
    exit 1
  fi

  status="$(git -C "$root" status --porcelain 2>/dev/null || true)"
  if [[ -n "$status" ]]; then
    echo "$flow_name must run from a linked worktree while the primary checkout has local changes." >&2
    echo "Keep the primary checkout clean on main, or switch into the worktree that holds the changes." >&2
    exit 1
  fi
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

host_uid() {
  id -u
}

host_gid() {
  id -g
}

runner_home_dir() {
  printf '/tmp/hotspot-home\n'
}

timestamp_run_id() {
  date +%Y%m%d-%H%M%S
}

source_root_from_current_worktree() {
  local root="$1"
  printf '%s\n' "$root"
}

resolved_git_ref() {
  local root="$1"
  local ref="$2"
  git -C "$root" rev-parse --short "$ref"
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
