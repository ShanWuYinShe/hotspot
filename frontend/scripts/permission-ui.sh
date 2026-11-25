#!/usr/bin/env bash
# 权限态 UI 门禁：/admin 头部与后台工具必须随登录/权限收敛。
# 断言：非管理员（QA）打开 /admin/users 时正文为「无访问权限」，岛标题为「管理后台」+「需要管理员权限」，
#       不出现「平台管理 / 返回概览 / 管理模块导航」；切换管理员后同一 URL 恢复模块语境与后台工具。
# 前置：dev 环境已启动且 seed 已注入；agent-browser 已安装。
#
# 用法：
#   bun run audit:permission
#   bun run audit:permission -- --base http://localhost:8899
# 输出：<仓库根>/.workwork/browser-check/permission-<时间戳>/（summary.txt）
# 退出码：任一断言失败 / 登录失败 / 后端不可达 → 1。

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib/qa-common.sh"
find_ab
find_node

BASE="http://localhost:8888"
SESSION="hotspot-audit-permission-$$"
while [ $# -gt 0 ]; do
  case "$1" in
    --base) BASE="$2"; shift ;;
    --session) SESSION="$2"; shift ;;
    --help|-h) sed -n '2,11p' "$0" | sed 's/^# *//'; exit 0 ;;
    *) echo "未知参数: $1（--help 查看用法）"; exit 2 ;;
  esac
  shift
done

require_backend
OUT="$(make_out_dir permission)"
SUMMARY="$OUT/summary.txt"
: > "$SUMMARY"

FAIL=0
PASS_COUNT=0
say() { echo "$1" | tee -a "$SUMMARY"; }
pass() { say "✓ $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { say "✗ $1"; FAIL=1; }

cleanup() { ab close >/dev/null 2>&1 || true; }
trap cleanup EXIT

# 当前页 /admin 权限态：岛文本 + 正文空态 + 后台工具存在性。
page_state() {
  ab eval '(() => {
    const t = document.body.innerText || "";
    const island = document.querySelector("[data-desktop-dynamic-island]");
    const nav = document.querySelector("[role=group][aria-label*=\"管理模块\"]");
    return JSON.stringify({
      island: island ? island.innerText.replace(/[ \t\n]+/g, " ").trim() : "",
      hasNoAccess: t.includes("无访问权限") ? "true" : "false",
      hasReturn: t.includes("返回概览") ? "true" : "false",
      hasNav: nav ? "true" : "false",
      hasUserManagement: t.includes("用户管理") ? "true" : "false"
    });
  })()' 2>/dev/null | tail -1
}

field() { qa_json_field "$1" "$2"; }

say "HotSpot 权限态 UI 门禁"
say "BASE=$BASE SESSION=$SESSION"
say "输出：$OUT"

ab viewport 1440 900 >/dev/null 2>&1

# ---- 1) 非管理员（QA 普通账号）：头部不得渲染平台管理语境与后台工具 ----
ensure_login >/dev/null 2>&1
if [ "${SCAN_LOGGED_IN:-0}" != "1" ]; then
  fail "QA 账号登录失败（先执行 ./scripts/seed-dev.sh）"
else
  pass "QA 普通账号登录"
  ab open "$BASE/admin/users" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  sleep 2.5
  STATE="$(page_state)"
  ISLAND="$(field "$STATE" island)"
  if [ "$(field "$STATE" hasNoAccess)" = "true" ]; then
    pass "非管理员：正文显示「无访问权限」"
  else
    fail "非管理员：正文未显示「无访问权限」（state=${STATE}）"
  fi
  case "$ISLAND" in
    *管理后台*需要管理员权限*) pass "非管理员：岛标题为「管理后台 / 需要管理员权限」" ;;
    *) fail "非管理员：岛文本不符合预期（island=${ISLAND}）" ;;
  esac
  case "$ISLAND" in
    *平台管理*) fail "非管理员：岛仍出现「平台管理」语境（island=${ISLAND}）" ;;
    *) pass "非管理员：岛不再出现「平台管理」" ;;
  esac
  if [ "$(field "$STATE" hasReturn)" = "true" ]; then
    fail "非管理员：仍显示「返回概览」"
  else
    pass "非管理员：不显示「返回概览」"
  fi
  if [ "$(field "$STATE" hasNav)" = "true" ]; then
    fail "非管理员：仍渲染「管理模块导航」"
  else
    pass "非管理员：不渲染「管理模块导航」"
  fi
  if [ "$(field "$STATE" hasUserManagement)" = "true" ]; then
    fail "非管理员：正文/头部仍出现「用户管理」模块名"
  else
    pass "非管理员：不出现「用户管理」模块名"
  fi
fi

# ---- 2) 切换管理员账号：同一 URL 恢复模块语境与后台工具 ----
ADMIN_PASSWORD="${QA_ADMIN_PASSWORD:-Admin12345678}"
qa_browser_signin "$QA_ADMIN_LOGIN_ID_DEFAULT" "$ADMIN_PASSWORD"
ADMIN_OK=0
for _ in $(seq 1 24); do
  raw="$(qa_poll_window_json 'window.__qaLogin' 2>/dev/null)"
  if [ -n "$raw" ] && [ "$(qa_json_code "$raw")" = "0" ]; then ADMIN_OK=1; break; fi
  sleep 0.5
done
if [ "$ADMIN_OK" != "1" ]; then
  fail "管理员账号登录失败（${QA_ADMIN_LOGIN_ID_DEFAULT}）"
else
  pass "管理员账号登录"
  ab open "$BASE/admin/users" >/dev/null 2>&1
  ab wait --load networkidle >/dev/null 2>&1
  sleep 2.5
  ADMIN_STATE="$(page_state)"
  ADMIN_ISLAND="$(field "$ADMIN_STATE" island)"
  if [ "$(field "$ADMIN_STATE" hasNoAccess)" = "false" ]; then
    pass "管理员：正文无「无访问权限」"
  else
    fail "管理员：正文仍显示「无访问权限」（state=${ADMIN_STATE}）"
  fi
  case "$ADMIN_ISLAND" in
    *用户管理*平台管理*返回概览*) pass "管理员：岛恢复「用户管理 / 平台管理 / 返回概览」" ;;
    *) fail "管理员：岛文本不符合预期（island=${ADMIN_ISLAND}）" ;;
  esac
  if [ "$(field "$ADMIN_STATE" hasNav)" = "true" ]; then
    pass "管理员：渲染「管理模块导航」"
  else
    fail "管理员：缺少「管理模块导航」"
  fi
fi

say ""
if [ "$FAIL" -eq 0 ]; then
  say "结果: PASS（通过 ${PASS_COUNT}）"
else
  say "结果: FAIL（通过 ${PASS_COUNT}）"
fi
say "报告目录：$OUT"
exit "$FAIL"
