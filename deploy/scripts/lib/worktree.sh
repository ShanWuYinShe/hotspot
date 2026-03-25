#!/usr/bin/env bash

acquire_worktree_metadata_lock() {
  local root="$1"
  acquire_lock "$root" git-worktree-metadata
}

deploy_worktrees_root() {
  local root="$1"
  printf '%s/worktrees\n' "$(deploy_tmp_root "$root")"
}

is_deploy_managed_worktree_path() {
  local root="$1"
  local path="$2"
  local managed_root
  managed_root="$(deploy_worktrees_root "$root")"
  case "$path" in
    "$managed_root"/*) return 0 ;;
    *) return 1 ;;
  esac
}

worktree_is_registered() {
  local root="$1"
  local target="$2"
  local line
  local candidate
  while IFS= read -r line; do
    if [[ "$line" == worktree\ * ]]; then
      candidate="${line#worktree }"
      if [[ "$candidate" == "$target" ]]; then
        return 0
      fi
    fi
  done < <(git -C "$root" worktree list --porcelain 2>/dev/null || true)
  return 1
}

release_worktree_dir() {
  local root="$1"
  local label="$2"
  printf '%s/%s\n' "$(deploy_worktrees_root "$root")" "$(sanitize_name "$label")"
}

ensure_release_worktree() {
  local root="$1"
  local ref="$2"
  local label="$3"
  local destination
  local metadata_lock
  destination="$(release_worktree_dir "$root" "$label")"
  metadata_lock="$(acquire_worktree_metadata_lock "$root")"
  if worktree_is_registered "$root" "$destination"; then
    echo "deploy-managed worktree is already registered: $destination" >&2
    release_lock "$metadata_lock"
    return 1
  fi
  if ! cleanup_path "$destination"; then
    release_lock "$metadata_lock"
    return 1
  fi
  if ! git -C "$root" worktree add --detach "$destination" "$ref" >/dev/null; then
    release_lock "$metadata_lock"
    return 1
  fi
  release_lock "$metadata_lock"
  printf '%s\n' "$destination"
}

remove_release_worktree() {
  local root="$1"
  local path="$2"
  local metadata_lock
  metadata_lock="$(acquire_worktree_metadata_lock "$root")"
  if git -C "$root" worktree remove --force "$path" >/dev/null 2>&1; then
    release_lock "$metadata_lock"
    return 0
  fi
  if ! is_deploy_managed_worktree_path "$root" "$path"; then
    echo "refusing fallback cleanup outside deploy-managed worktrees: $path" >&2
    release_lock "$metadata_lock"
    return 1
  fi
  if worktree_is_registered "$root" "$path"; then
    echo "git worktree remove failed while $path is still registered; refusing direct cleanup and prune." >&2
    release_lock "$metadata_lock"
    return 1
  fi
  echo "git worktree remove failed for $path, but it is no longer registered; cleaning leftover deploy-managed path only" >&2
  if ! cleanup_path "$path"; then
    release_lock "$metadata_lock"
    return 1
  fi
  release_lock "$metadata_lock"
}
