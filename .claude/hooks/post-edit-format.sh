#!/usr/bin/env bash
set -euo pipefail

file_path=$(python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('tool_input', {}).get('file_path', ''))
except:
    print('')
")

[[ -z "$file_path" || "$file_path" == "null" ]] && exit 0

cd "${CLAUDE_PROJECT_DIR:-.}"

case "$file_path" in
  */backend-go/*)
    case "$file_path" in
      *.go) command -v gofmt >/dev/null 2>&1 && gofmt -w "$file_path" 2>/dev/null || true ;;
    esac
    ;;
  */frontend/*)
    case "$file_path" in
      *.ts|*.tsx|*.js|*.jsx|*.svelte|*.css|*.html|*.json|*.yaml|*.yml|*.md)
        if [ -f "frontend/node_modules/.bin/prettier" ]; then
          frontend/node_modules/.bin/prettier --write "$file_path" 2>/dev/null || true
        fi
        ;;
    esac
    ;;
esac

exit 0
