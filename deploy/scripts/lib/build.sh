#!/usr/bin/env bash

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
  printf '%s\n' 'rm -rf /tmp/workspace && mkdir -p /tmp/workspace/frontend_svelte && cp -a /workspace/frontend_svelte/. /tmp/workspace/frontend_svelte/ && cd /tmp/workspace/frontend_svelte && CI=true pnpm install --frozen-lockfile && CI=true pnpm build && CI=true pnpm prune --prod && cp -R node_modules /out/node_modules && cp package.json pnpm-lock.yaml /out/ && rm -rf /out/build && cp -R build /out/build'
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
