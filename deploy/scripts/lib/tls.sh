#!/usr/bin/env bash

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
