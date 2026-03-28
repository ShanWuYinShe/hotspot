#!/usr/bin/env bash

default_postgres_image() {
  printf 'postgres:18.3-alpine3.23\n'
}

default_redis_image() {
  printf 'redis:8.6.2-alpine3.23\n'
}

default_preview_minio_image() {
  printf 'minio/minio:RELEASE.2025-09-07T16-13-09Z\n'
}

default_preview_nginx_image() {
  printf 'nginx:1.28.3-alpine3.23\n'
}

default_golang_base_image() {
  printf 'golang:1.26.1-alpine3.23\n'
}

default_node_base_image() {
  printf 'node:24.14.0-alpine3.23\n'
}

default_alpine_base_image() {
  printf 'alpine:3.23.3\n'
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

export_verify_image_defaults() {
  export HOTSPOT_VERIFY_POSTGRES_IMAGE="${HOTSPOT_VERIFY_POSTGRES_IMAGE:-$(default_postgres_image)}"
  export HOTSPOT_VERIFY_REDIS_IMAGE="${HOTSPOT_VERIFY_REDIS_IMAGE:-$(default_redis_image)}"
  export HOTSPOT_VERIFY_MINIO_IMAGE="${HOTSPOT_VERIFY_MINIO_IMAGE:-$(default_preview_minio_image)}"
}

export_preview_image_defaults() {
  export HOTSPOT_PREVIEW_POSTGRES_IMAGE="${HOTSPOT_PREVIEW_POSTGRES_IMAGE:-$(default_postgres_image)}"
  export HOTSPOT_PREVIEW_REDIS_IMAGE="${HOTSPOT_PREVIEW_REDIS_IMAGE:-$(default_redis_image)}"
  export HOTSPOT_PREVIEW_MINIO_IMAGE="${HOTSPOT_PREVIEW_MINIO_IMAGE:-$(default_preview_minio_image)}"
  export HOTSPOT_PREVIEW_NGINX_IMAGE="${HOTSPOT_PREVIEW_NGINX_IMAGE:-$(default_preview_nginx_image)}"
}

remove_preview_images() {
  docker image rm -f "$(preview_backend_image)" >/dev/null 2>&1 || true
  docker image rm -f "$(preview_frontend_image)" >/dev/null 2>&1 || true
}
