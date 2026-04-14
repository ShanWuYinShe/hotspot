#!/usr/bin/env bash
set -euo pipefail

input=$(cat)

tool_name=$(python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('tool_name', ''))
except:
    print('')
" <<< "$input")

if [[ "$tool_name" != "Edit" && "$tool_name" != "Write" && "$tool_name" != "MultiEdit" ]]; then
    exit 0
fi

file_path=$(python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('tool_input', {}).get('file_path', ''))
except:
    print('')
" <<< "$input")

[[ -z "$file_path" || "$file_path" == "null" ]] && exit 0

project_dir="${CLAUDE_PROJECT_DIR:-.}"

case "$file_path" in
  "$project_dir"/spec/*|*/spec/*)
    echo "spec/ 是只读的 API 契约目录，不应直接修改。如需修改请先确认用户明确要求。" >&2
    exit 2
    ;;
esac

exit 0
