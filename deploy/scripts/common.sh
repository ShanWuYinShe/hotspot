#!/usr/bin/env bash

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-'
}

detect_env_name() {
  local root="$1"
  local value="${HOTSPOT_ENV_NAME:-}"
  if [[ -n "$value" ]]; then
    sanitize_name "$value"
    return
  fi
  if command -v git >/dev/null 2>&1; then
    value="$(git -C "$root" branch --show-current 2>/dev/null || true)"
  fi
  if [[ -z "$value" ]]; then
    value="default"
  fi
  sanitize_name "$value"
}

derive_runtime_ports() {
  local env_name="$1"
  if [[ "$env_name" == "default" ]]; then
    printf '8088 8443\n'
    return
  fi
  local hash
  hash="$(printf '%s' "$env_name" | cksum | awk '{print $1}')"
  local offset=$((hash % 1000))
  printf '%d %d\n' "$((18080 + offset))" "$((28080 + offset))"
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
