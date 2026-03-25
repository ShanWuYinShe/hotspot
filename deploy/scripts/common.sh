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

default_postgres_image() {
  printf 'postgres:17.9-alpine3.23\n'
}

default_redis_image() {
  printf 'redis:7.4.8-alpine3.21\n'
}

default_preview_minio_image() {
  printf 'minio/minio:RELEASE.2025-09-07T16-13-09Z\n'
}

default_preview_minio_mc_image() {
  printf 'minio/mc:RELEASE.2025-08-13T08-35-41Z\n'
}

default_preview_nginx_image() {
  printf 'nginx:1.28.3-alpine3.23\n'
}

default_golang_base_image() {
  printf 'golang:1.26.1-alpine3.23\n'
}

default_node_base_image() {
  printf 'node:22.22.1-alpine3.23\n'
}

default_alpine_base_image() {
  printf 'alpine:3.23.3\n'
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

ensure_writable_dir() {
  local path="$1"
  if [[ ! -w "$path" ]]; then
    echo "deploy path is not writable by the current user: $path" >&2
    echo "fix the ownership of this tmp/deploy path before rerunning the flow." >&2
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

preview_tls_cert_dir() {
  local root="$1"
  printf '%s/tmp/certs\n' "$root"
}

preview_default_tls_cert_path() {
  local root="$1"
  printf '%s/localhost.crt\n' "$(preview_tls_cert_dir "$root")"
}

preview_default_tls_key_path() {
  local root="$1"
  printf '%s/localhost.key\n' "$(preview_tls_cert_dir "$root")"
}

tls_pair_exists() {
  local cert_path="$1"
  local key_path="$2"
  [[ -f "$cert_path" && -f "$key_path" ]]
}

generate_local_tls_cert_pair() {
  local cert_path="$1"
  local key_path="$2"
  local cert_dir
  local key_dir
  local openssl_config_path
  cert_dir="$(dirname "$cert_path")"
  key_dir="$(dirname "$key_path")"
  ensure_dir "$cert_dir"
  ensure_writable_dir "$cert_dir"
  if [[ "$key_dir" != "$cert_dir" ]]; then
    ensure_dir "$key_dir"
    ensure_writable_dir "$key_dir"
  fi

  if command -v mkcert >/dev/null 2>&1; then
    echo "Generating local TLS certificate with mkcert: $cert_path" >&2
    if mkcert -cert-file "$cert_path" -key-file "$key_path" localhost 127.0.0.1 ::1 >/dev/null 2>&1; then
      return 0
    fi
    echo "mkcert failed; falling back to an openssl self-signed certificate." >&2
  fi

  if ! command -v openssl >/dev/null 2>&1; then
    return 1
  fi

  openssl_config_path="$cert_dir/openssl-localhost.cnf"
  cat >"$openssl_config_path" <<'EOF'
[req]
default_bits = 2048
distinguished_name = req_distinguished_name
prompt = no
x509_extensions = v3_req

[req_distinguished_name]
CN = localhost

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

  echo "Generating self-signed TLS certificate with openssl: $cert_path" >&2
  if openssl req -x509 -nodes -newkey rsa:2048 -sha256 -days 365 -keyout "$key_path" -out "$cert_path" -config "$openssl_config_path" -extensions v3_req >/dev/null 2>&1; then
    rm -f "$openssl_config_path"
    return 0
  fi

  rm -f "$openssl_config_path"
  return 1
}

resolve_preview_tls_paths() {
  local root="$1"
  local primary_root="$2"
  local requested_cert_path="$3"
  local requested_key_path="$4"
  local default_cert_path
  local default_key_path
  local fallback_cert_path=""
  local fallback_key_path=""
  local cert_path
  local key_path

  default_cert_path="$(preview_default_tls_cert_path "$root")"
  default_key_path="$(preview_default_tls_key_path "$root")"

  if [[ -n "$requested_cert_path" || -n "$requested_key_path" ]]; then
    cert_path="${requested_cert_path:-$default_cert_path}"
    key_path="${requested_key_path:-$default_key_path}"
    if tls_pair_exists "$cert_path" "$key_path"; then
      printf '%s\t%s\n' "$cert_path" "$key_path"
      return 0
    fi

    cat >&2 <<EOF
Configured TLS certificate or key not found.
Expected:
  HOTSPOT_TLS_CERT_PATH=$cert_path
  HOTSPOT_TLS_KEY_PATH=$key_path

Either fix the explicit paths or unset HOTSPOT_TLS_CERT_PATH/HOTSPOT_TLS_KEY_PATH to let preview-up prepare default certs under:
  $default_cert_path
  $default_key_path
EOF
    return 1
  fi

  if tls_pair_exists "$default_cert_path" "$default_key_path"; then
    printf '%s\t%s\n' "$default_cert_path" "$default_key_path"
    return 0
  fi

  if [[ -n "$primary_root" && "$primary_root" != "$root" ]]; then
    fallback_cert_path="$(preview_default_tls_cert_path "$primary_root")"
    fallback_key_path="$(preview_default_tls_key_path "$primary_root")"
    if tls_pair_exists "$fallback_cert_path" "$fallback_key_path"; then
      printf '%s\t%s\n' "$fallback_cert_path" "$fallback_key_path"
      return 0
    fi
  fi

  if generate_local_tls_cert_pair "$default_cert_path" "$default_key_path"; then
    printf '%s\t%s\n' "$default_cert_path" "$default_key_path"
    return 0
  fi

  cat >&2 <<EOF
TLS certificate or key not found, and automatic generation failed.
Checked:
  HOTSPOT_TLS_CERT_PATH=$default_cert_path
  HOTSPOT_TLS_KEY_PATH=$default_key_path
EOF
  if [[ -n "$fallback_cert_path" && -n "$fallback_key_path" ]]; then
    cat >&2 <<EOF
Fallback checked:
  HOTSPOT_TLS_CERT_PATH=$fallback_cert_path
  HOTSPOT_TLS_KEY_PATH=$fallback_key_path
EOF
  fi
  cat >&2 <<'EOF'
Install mkcert or openssl, or set HOTSPOT_TLS_CERT_PATH/HOTSPOT_TLS_KEY_PATH to an existing certificate pair.
EOF
  return 1
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

timestamp_run_id() {
  date +%Y%m%d-%H%M%S
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

source_root_from_current_worktree() {
  local root="$1"
  printf '%s\n' "$root"
}

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

common_docker_build_args() {
  local -a build_args=()
  build_args+=(--build-arg "HOTSPOT_GOLANG_BASE_IMAGE=${HOTSPOT_GOLANG_BASE_IMAGE:-$(default_golang_base_image)}")
  build_args+=(--build-arg "HOTSPOT_NODE_BASE_IMAGE=${HOTSPOT_NODE_BASE_IMAGE:-$(default_node_base_image)}")
  build_args+=(--build-arg "HOTSPOT_ALPINE_BASE_IMAGE=${HOTSPOT_ALPINE_BASE_IMAGE:-$(default_alpine_base_image)}")
  [[ -n "${HOTSPOT_ALPINE_REPO:-}" ]] && build_args+=(--build-arg "HOTSPOT_ALPINE_REPO=$HOTSPOT_ALPINE_REPO")
  printf '%s\0' "${build_args[@]}"
}

ensure_runner_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local -a build_args=()
  if docker image inspect "$image" >/dev/null 2>&1; then
    return 0
  fi
  while IFS= read -r -d '' arg; do
    build_args+=("$arg")
  done < <(common_docker_build_args)
  if ((${#build_args[@]} > 0)); then
    docker build "${build_args[@]}" -t "$image" -f "$dockerfile" "$context"
  else
    docker build -t "$image" -f "$dockerfile" "$context"
  fi
}

build_app_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local -a build_args=()
  while IFS= read -r -d '' arg; do
    build_args+=("$arg")
  done < <(common_docker_build_args)
  if ((${#build_args[@]} > 0)); then
    docker build "${build_args[@]}" -t "$image" -f "$dockerfile" "$context"
  else
    docker build -t "$image" -f "$dockerfile" "$context"
  fi
}

backend_build_command() {
  printf '%s\n' 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/backend-go && cp -a /workspace/backend-go/. /tmp/workspace/backend-go/ && cd /tmp/workspace/backend-go && go build -o /out/backend-go ./cmd/server'
}

frontend_build_command() {
  printf '%s\n' 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/frontend_svelte && cp -a /workspace/frontend_svelte/. /tmp/workspace/frontend_svelte/ && cd /tmp/workspace/frontend_svelte && CI=true pnpm install --frozen-lockfile && CI=true pnpm build && cp package.json pnpm-lock.yaml /out/ && rm -rf /out/build && cp -R build /out/build'
}

run_backend_artifact_build() {
  local log_path="$1"
  local source_root="$2"
  local artifact_dir="$3"
  local runner_image="$4"
  run_and_capture "$log_path" docker run --rm \
    --user "$(host_uid):$(host_gid)" \
    -e HOME="$(runner_home_dir)" \
    -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
    -v "$source_root:/workspace:ro" \
    -v "$artifact_dir:/out" \
    "$runner_image" \
    sh -c "$(backend_build_command)"
}

run_frontend_artifact_build() {
  local log_path="$1"
  local source_root="$2"
  local artifact_dir="$3"
  local runner_image="$4"
  run_and_capture "$log_path" docker run --rm \
    --user "$(host_uid):$(host_gid)" \
    -e HOME="$(runner_home_dir)" \
    -e XDG_CACHE_HOME="$(runner_home_dir)/.cache" \
    -v "$source_root:/workspace:ro" \
    -v "$artifact_dir:/out" \
    "$runner_image" \
    sh -c "$(frontend_build_command)"
}

export_verify_image_defaults() {
  export HOTSPOT_VERIFY_POSTGRES_IMAGE="${HOTSPOT_VERIFY_POSTGRES_IMAGE:-$(default_postgres_image)}"
  export HOTSPOT_VERIFY_REDIS_IMAGE="${HOTSPOT_VERIFY_REDIS_IMAGE:-$(default_redis_image)}"
}

export_preview_image_defaults() {
  export HOTSPOT_PREVIEW_POSTGRES_IMAGE="${HOTSPOT_PREVIEW_POSTGRES_IMAGE:-$(default_postgres_image)}"
  export HOTSPOT_PREVIEW_REDIS_IMAGE="${HOTSPOT_PREVIEW_REDIS_IMAGE:-$(default_redis_image)}"
  export HOTSPOT_PREVIEW_MINIO_IMAGE="${HOTSPOT_PREVIEW_MINIO_IMAGE:-$(default_preview_minio_image)}"
  export HOTSPOT_PREVIEW_MINIO_MC_IMAGE="${HOTSPOT_PREVIEW_MINIO_MC_IMAGE:-$(default_preview_minio_mc_image)}"
  export HOTSPOT_PREVIEW_NGINX_IMAGE="${HOTSPOT_PREVIEW_NGINX_IMAGE:-$(default_preview_nginx_image)}"
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
