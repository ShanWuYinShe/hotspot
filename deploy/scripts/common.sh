#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/env.sh"
source "$SCRIPT_DIR/lib/images.sh"
source "$SCRIPT_DIR/lib/fs.sh"
source "$SCRIPT_DIR/lib/tls.sh"
source "$SCRIPT_DIR/lib/worktree.sh"
source "$SCRIPT_DIR/lib/build.sh"

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

s3_healthcheck_path() {
  printf '/s3/minio/health/live\n'
}

s3_healthcheck_ok() {
  local base_url="$1"
  local code
  code="$(curl -sS -o /dev/null -w "%{http_code}" "${base_url}$(s3_healthcheck_path)" || true)"
  [[ "$code" == "200" ]]
}
