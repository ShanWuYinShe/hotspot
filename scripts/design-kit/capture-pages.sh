#!/bin/bash
# 重新捕获 Web 端 390×844 页面截图到 docs/design-kit/pages。
# 前置：docker compose 本地环境已启动（http://localhost:8888），浏览器已有登录态（QA/管理员）。
# 用法：bash scripts/design-kit/capture-pages.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="$ROOT/docs/design-kit/pages"
mkdir -p "$OUT"

agent-browser open http://localhost:8888 >/dev/null 2>&1
agent-browser set viewport 390 844 >/dev/null 2>&1
agent-browser set media light >/dev/null 2>&1
agent-browser wait 1200 >/dev/null 2>&1

capture() {
  local route="$1"
  local name="$2"
  agent-browser open "http://localhost:8888$route" >/dev/null 2>&1
  agent-browser wait 1400 >/dev/null 2>&1
  agent-browser screenshot "$OUT/web-$name.png" >/dev/null 2>&1
  echo "captured web-$name.png"
}

capture / home
capture /nearby nearby
capture /nearby/people nearby-people
capture /nearby/moments nearby-moments
capture /nearby/groups nearby-groups
capture /messages messages
capture /notify notify
capture /me me
capture /settings settings
capture /search search
capture /me/edit me-edit
capture /me/relationships relationships
capture /me/moments my-moments
capture /me/groups my-groups
capture /me/reports my-reports
capture /map map
capture /groups/group_seed_1 group-detail
capture /moments/moment_seed_01 moment-detail
capture /users/user_seed_lumos user-profile
capture /admin admin

agent-browser close >/dev/null 2>&1 || true
echo "done: $(ls -1 "$OUT" | grep -c '^web-') web screenshots"

