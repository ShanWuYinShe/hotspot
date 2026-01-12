#!/bin/sh
# Guard: forbid ambiguous bare `id` fields across spec/backend/frontend/types.
#
# Why:
# - Bare `id` is ambiguous (entity type unclear) and leads to spec/code drift.
# - HotSpot convention: use explicit `xxx_id` (e.g. user_id, moment_id, group_chat_id, message_id).
#
# Scope (default):
# - API spec (Cursor rule copy): .cursor/rules/api_spec.mdc
# - Backend Rust DTOs:          backend/src/dto/**/*.rs
# - Frontend API types:         frontend/app/types/**/*.ts
#
# Policy (conservative):
# - Disallow object/interface fields named exactly `id` (e.g. `id: string`, `pub id: String`, `"id": ...`)
# - Disallow common API payload patterns that include an object-literal key named exactly `id`
#
# Allowlist:
# - Keep allow rules targeted. Prefer excluding a specific file or an exact-shaped false-positive line.
#
# Usage:
#   sh hotspot/scripts/guard-no-bare-id.sh
#
# Exit codes:
#   0 = OK
#   1 = Violations found

set -eu

# Resolve paths relative to this script's location so it works no matter where it's run from.
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

# ---- Paths to scan (keep these narrow to avoid false positives) ----
SPEC_PATH="$REPO_ROOT/.cursor/rules/api_spec.mdc"
BACKEND_DTO_GLOB="$REPO_ROOT/backend/src/dto"
FRONTEND_TYPES_GLOB="$REPO_ROOT/frontend/app/types"

# Optional: scan Vue/TS call sites for object literal keys named exactly `id:`
FRONTEND_APP_GLOB="$REPO_ROOT/frontend/app"

# ---- Allowlist: files to exclude entirely (rare) ----
# One path per line, absolute (or exactly as printed by grep).
EXCLUDE_FILES=""

# ---- Allowlist: regex patterns (grep -E) to ignore specific false positives ----
# One regex per line. Keep these narrow and preferably path-anchored.
ALLOW_FILTERS=""

fail=0

say() {
  printf "%s\n" "$*"
}

# Portable recursive grep (prints file:line:content)
# -R recursive, -n line numbers, -E extended regex
run_grep() {
  # $1: regex, $2..: paths
  pattern="$1"
  shift
  # shellcheck disable=SC2048,SC2086
  grep -R -n -E -- "$pattern" "$@" 2>/dev/null || true
}

# Apply file excludes (exact match on "path:" prefix)
filter_excludes() {
  in="$1"
  out="$in"

  if [ -n "$EXCLUDE_FILES" ]; then
    # Iterate lines without spawning a subshell that would lose $out via pipe.
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      out="$(printf "%s\n" "$out" | grep -v -F -- "$f:" || true)"
    done <<EOF
+$EXCLUDE_FILES
+EOF
  fi

  printf "%s\n" "$out"
}

# Apply allow-filters (regex filters)
filter_allow() {
  in="$1"
  out="$in"

  if [ -n "$ALLOW_FILTERS" ]; then
    while IFS= read -r pat; do
      [ -n "$pat" ] || continue
      out="$(printf "%s\n" "$out" | grep -v -E -- "$pat" || true)"
    done <<EOF
+$ALLOW_FILTERS
+EOF
  fi

  printf "%s\n" "$out"
}

# Shared runner: grep -> exclude -> allow -> print if non-empty
check() {
  title="$1"
  regex="$2"
  shift 2

  # shellcheck disable=SC2086
  matches="$(run_grep "$regex" "$@")"
  matches="$(filter_excludes "$matches")"
  matches="$(filter_allow "$matches")"

  if [ -n "$matches" ]; then
    fail=1
    say "Violation: $title"
    say "  regex: $regex"
    say ""
    say "$matches"
    say ""
  fi
}

say "[guard-no-bare-id] Scanning for ambiguous bare \`id\` fields..."
say ""

# --------------------
# 1) Spec: forbid `id:` fields in TypeScript code blocks / interfaces
# --------------------
if [ -f "$SPEC_PATH" ]; then
  check "Spec must not define bare id fields (use xxx_id instead)" \
    '(^|[^a-zA-Z0-9_])id[[:space:]]*:[[:space:]]*[^;]+;?' \
    "$SPEC_PATH"
else
  say "[guard-no-bare-id] Note: spec file not found at $SPEC_PATH (skipping)"
  say ""
fi

# --------------------
# 2) Backend Rust DTOs: forbid `pub id: ...` in DTO structs + serde rename traps
# --------------------
if [ -d "$BACKEND_DTO_GLOB" ]; then
  check "Backend DTOs must not expose bare `id` fields (use xxx_id)" \
    '^[[:space:]]*pub[[:space:]]+id[[:space:]]*:' \
    "$BACKEND_DTO_GLOB"

  check "Backend DTOs must not serialize/deserialize as \"id\" (use xxx_id)" \
    'serde[[:space:]]*\([[:space:]]*rename[[:space:]]*=[[:space:]]*"id"[[:space:]]*\)' \
    "$BACKEND_DTO_GLOB"
else
  say "[guard-no-bare-id] Note: backend DTO directory not found at $BACKEND_DTO_GLOB (skipping)"
  say ""
fi

# --------------------
# 3) Frontend API types: forbid `id:` fields
# --------------------
if [ -d "$FRONTEND_TYPES_GLOB" ]; then
  check "Frontend API types must not define bare `id` fields (use xxx_id)" \
    '(^|[^a-zA-Z0-9_])id[[:space:]]*\??[[:space:]]*:[[:space:]]*' \
    "$FRONTEND_TYPES_GLOB"
else
  say "[guard-no-bare-id] Note: frontend types directory not found at $FRONTEND_TYPES_GLOB (skipping)"
  say ""
fi

# --------------------
# 4) Frontend call sites (optional): restrict to object-literal field key `id:`
# --------------------
if [ -d "$FRONTEND_APP_GLOB" ]; then
  # Only match object literal keys named exactly `id:` at the start of a key/value pair.
  # This avoids false positives like:
  #   - function params: `(id: string)`
  #   - object index access: `obj[id]`
  #   - helper functions: `get(id)`
  check "Frontend must not construct object literals with key `id:` for API payloads (use xxx_id)" \
    '^[^/]*\{[^}]*\bid[[:space:]]*:[[:space:]]*' \
    "$FRONTEND_APP_GLOB"
fi

if [ "$fail" -ne 0 ]; then
  say "[guard-no-bare-id] FAILED"
  say ""
  say "Fix guidance:"
  say "  - Replace bare `id` with explicit `xxx_id` everywhere."
  say "    Examples:"
  say "      user_id (number), moment_id/comment_id/group_chat_id/private_chat_id/message_id (string UUID)"
  say ""
  say "  - If a field truly represents an identifier, name it by its domain entity:"
  say "      e.g. session_id, report_id, invite_id, application_id, device_token_id, history_id, announcement_id"
  say ""
  say "  - Avoid compat aliases like serde rename(\"id\"); they reintroduce ambiguity."
  exit 1
fi

say "[guard-no-bare-id] OK"
exit 0
