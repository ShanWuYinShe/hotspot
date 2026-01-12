#!/bin/sh
# Guard: prevent hardcoded API path prefixes in HarmonyOS ETS sources.
#
# Policy:
# - Disallow hardcoded "/api/v1/" anywhere in Harmony app ETS code, except:
#   - hotspot/harmonyapp/entry/src/main/ets/core/config/ApiEndpoints.ets
# - Disallow hardcoded "/api/<number>/" to avoid drift when bumping API version.
#
# Usage:
#   sh hotspot/scripts/guard-api-paths.sh
#
# Exit codes:
#   0 = OK
#   1 = Violations found

set -eu

PROJECT_ROOT="hotspot"
ETS_GLOB="$PROJECT_ROOT/harmonyapp/entry/src/main/ets"

ALLOW_PATH="$ETS_GLOB/core/config/ApiEndpoints.ets"

# These patterns are deliberately conservative:
# - "/api/v1/" catches the common hardcode
# - "/api/v[0-9]+/" catches any version hardcode like v2/v3...
PATTERN_V1='/api/v1/'
PATTERN_ANY_VERSION='/api/v[0-9]+/'

fail=0

say() {
  printf "%s\n" "$*"
}

# Portable grep runner:
# - Use -n for line numbers
# - Use -E for extended regex
# - Use -R for recursion
# - Use -- to terminate options (safe for weird paths)
run_grep() {
  # shellcheck disable=SC2048,SC2086
  grep -R -n -E -- "$1" "$2" 2>/dev/null || true
}

say "[guard-api-paths] Scanning HarmonyOS ETS sources for hardcoded API prefixes..."
say "  root: $ETS_GLOB"
say "  allow: $ALLOW_PATH"
say ""

# 1) Forbid exact /api/v1/ outside allowlist.
# We search everything under ETS, then filter out allow file.
matches_v1="$(run_grep "$(printf "%s" "$PATTERN_V1" | sed 's/[.[\*^$(){}?+|/]/\\&/g')" "$ETS_GLOB" \
  | grep -v -F -- "$ALLOW_PATH:" \
  || true)"

if [ -n "$matches_v1" ]; then
  fail=1
  say "Violation: hardcoded \"$PATTERN_V1\" found outside ApiEndpoints registry:"
  say "$matches_v1"
  say ""
fi

# 2) Forbid any /api/vN/ (N number) outside allowlist.
matches_any_version="$(run_grep "$PATTERN_ANY_VERSION" "$ETS_GLOB" \
  | grep -v -F -- "$ALLOW_PATH:" \
  || true)"

if [ -n "$matches_any_version" ]; then
  fail=1
  say "Violation: hardcoded versioned API prefix matching \"$PATTERN_ANY_VERSION\" found outside ApiEndpoints registry:"
  say "$matches_any_version"
  say ""
fi

if [ "$fail" -ne 0 ]; then
  say "[guard-api-paths] FAILED"
  say ""
  say "Fix guidance:"
  say "  - Replace hardcoded strings like \"/api/v1/...\" with values from:"
  say "      $ALLOW_PATH"
  say "  - If you need a new endpoint, add it to ApiEndpoints.ets ONLY if it exists in api_spec.mdc."
  exit 1
fi

say "[guard-api-paths] OK"
exit 0
