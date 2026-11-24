#!/usr/bin/env bash
# 列表键盘导航门禁：消息会话列表 ↑↓、搜索框 ↓ 进首行、/notify 列表 ↑↓。
# 前置：dev 环境已启动且 seed 已注入；agent-browser 已安装；frontend 已 bun install。
#
# 用法：
#   bun run audit:keyboard
#   bun run audit:keyboard -- --base http://localhost:8899
# 输出：<仓库根>/.workwork/browser-check/keyboard-<时间戳>/（summary.txt）
# 退出码：任一断言失败、未登录、列表不足 2 行（seed 缺失）→ 1。

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/qa-common.sh"
find_ab
find_node

BASE="${KEYBOARD_BASE:-http://localhost:8888}"
SESSION="${KEYBOARD_SESSION:-hotspot-audit-keyboard-$$}"
while [ $# -gt 0 ]; do
  case "${1}" in
    --base) BASE="${2:-${BASE}}"; shift ;;
    --session) SESSION="${2:-${SESSION}}"; shift ;;
    --help|-h) sed -n '2,9p' "${0}" | sed 's/^# *//'; exit 0 ;;
    *) echo "未知参数: ${1}（--help 查看用法）"; exit 2 ;;
  esac
  shift
done

require_backend
OUT="$(make_out_dir keyboard)"
SUMMARY="${OUT}/summary.txt"
: > "${SUMMARY}"

FAIL=0
PASS_COUNT=0
SKIP_COUNT=0
say() { echo "${1}" | tee -a "${SUMMARY}"; }
pass() { say "✓ ${1}"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { say "✗ ${1}"; FAIL=1; }
skip() { say "↷ ${1}"; SKIP_COUNT=$((SKIP_COUNT + 1)); }

cleanup() { ab close >/dev/null 2>&1 || true; }
trap cleanup EXIT

# 当前焦点元素的读屏名（无 aria-label 时为空）。
active_label() {
  local raw
  raw="$(ab eval 'JSON.stringify({active: (() => { const el = document.activeElement; return el ? (el.getAttribute("aria-label") || el.textContent || "").trim().slice(0, 40) : ""; })()})' 2>/dev/null | tail -1)"
  qa_json_field "${raw}" active
}

# 当前页列表行：数量 + 首行/第二行读屏名。
rows_json() {
  ab eval '(() => {
    const rs = [...document.querySelectorAll("[data-list-item] button[data-cover]")];
    return JSON.stringify({
      count: String(rs.length),
      first: rs[0]?.getAttribute("aria-label") || "",
      second: rs[1]?.getAttribute("aria-label") || ""
    });
  })()' 2>/dev/null | tail -1
}

# 搜索页结果候选（与 useSearchPageData.searchNavCandidates 同口径：
# 排除 header / tablist / input-surface，只取可见且非 aria-hidden 的 a/button/[data-cover]）。
search_rows_json() {
  ab eval '(() => {
    const root = document.querySelector("[data-search-root]");
    if (!root) return JSON.stringify({ count: "0", first: "", second: "" });
    const els = [...root.querySelectorAll("a[href], button:not([disabled]), [data-cover]")].filter((el) =>
      !el.closest("header") &&
      !el.closest("[role=\"tablist\"], [data-tab-bar], .input-surface") &&
      el.offsetParent !== null &&
      !el.hasAttribute("aria-hidden")
    );
    const label = (el) => (el.getAttribute("aria-label") || el.textContent || "").trim().slice(0, 40);
    return JSON.stringify({ count: String(els.length), first: els[0] ? label(els[0]) : "", second: els[1] ? label(els[1]) : "" });
  })()' 2>/dev/null | tail -1
}

# 聚焦搜索框（返回 ok=false 表示页面没有搜索框）。
focus_search() {
  ab eval '(() => {
    const s = document.querySelector("input[data-search-input]");
    if (!s) return JSON.stringify({ ok: "false" });
    s.focus();
    return JSON.stringify({ ok: "true" });
  })()' 2>/dev/null | tail -1
}

say "HotSpot 列表键盘导航门禁"
say "BASE=${BASE} SESSION=${SESSION}"
say "输出：${OUT}"

ab open "${BASE}/" >/dev/null 2>&1
ensure_login
if [ "${SCAN_LOGGED_IN:-0}" != "1" ]; then
  fail "QA 账号登录失败（先执行 ./scripts/seed-dev.sh 注入种子数据）"
  say "结果: FAIL"
  exit 1
fi
say "登录：QA 账号已登录"

# ---- 1) 消息会话列表：搜索框 ↓ 进首行、↓/↑ 逐项移动 ----
ab open "${BASE}/messages" >/dev/null 2>&1
COUNT=0
WAIT=0
while [ "${WAIT}" -lt 25 ]; do
  RAW="$(rows_json)"
  COUNT="$(qa_json_field "${RAW}" count)"
  [ "${COUNT:-0}" -ge 2 ] && break
  sleep 0.4
  WAIT=$((WAIT + 1))
done
FIRST="$(qa_json_field "${RAW}" first)"
SECOND="$(qa_json_field "${RAW}" second)"
if [ "${COUNT:-0}" -lt 2 ]; then
  skip "/messages 会话行不足 2（seed 会话缺失），跳过"
else
  FOCUS="$(focus_search)"
  if [ "$(qa_json_field "${FOCUS}" ok)" != "true" ]; then
    fail "/messages 找不到搜索会话输入框"
  else
    BEFORE="$(active_label)"
    ab press ArrowDown >/dev/null 2>&1
    AFTER_DOWN="$(active_label)"
    if [ -n "${FIRST}" ] && [ "${AFTER_DOWN}" = "${FIRST}" ]; then
      pass "搜索框 ↓ 进入首行（${FIRST}）"
    else
      fail "搜索框 ↓ 未进入首行（before=${BEFORE} after=${AFTER_DOWN} first=${FIRST}）"
    fi
    ab press ArrowDown >/dev/null 2>&1
    AFTER_DOWN2="$(active_label)"
    if [ -n "${SECOND}" ] && [ "${AFTER_DOWN2}" = "${SECOND}" ]; then
      pass "会话列表 ↓ 移到第二行（${SECOND}）"
    else
      fail "会话列表 ↓ 未移到第二行（after=${AFTER_DOWN2} second=${SECOND}）"
    fi
    ab press ArrowUp >/dev/null 2>&1
    AFTER_UP="$(active_label)"
    if [ "${AFTER_UP}" = "${FIRST}" ]; then
      pass "会话列表 ↑ 回到首行"
    else
      fail "会话列表 ↑ 未回到首行（after=${AFTER_UP} first=${FIRST}）"
    fi
  fi
fi

# ---- 2) /notify 列表：行内 ↓/↑ 逐项移动 ----
ab open "${BASE}/notify" >/dev/null 2>&1
COUNT=0
WAIT=0
while [ "${WAIT}" -lt 25 ]; do
  RAW="$(rows_json)"
  COUNT="$(qa_json_field "${RAW}" count)"
  [ "${COUNT:-0}" -ge 2 ] && break
  sleep 0.4
  WAIT=$((WAIT + 1))
done
FIRST="$(qa_json_field "${RAW}" first)"
SECOND="$(qa_json_field "${RAW}" second)"
if [ "${COUNT:-0}" -lt 2 ]; then
  skip "/notify 列表行不足 2（通知缺失），跳过"
else
  ab eval '(() => { const rs = [...document.querySelectorAll("[data-list-item] button[data-cover]")]; rs[0]?.focus(); return true; })()' >/dev/null 2>&1
  ab press ArrowDown >/dev/null 2>&1
  AFTER="$(active_label)"
  if [ -n "${SECOND}" ] && [ "${AFTER}" = "${SECOND}" ]; then
    pass "/notify ↓ 移到第二行（${SECOND}）"
  else
    fail "/notify ↓ 未移动（after=${AFTER} second=${SECOND}）"
  fi
  ab press ArrowUp >/dev/null 2>&1
  BACK="$(active_label)"
  if [ "${BACK}" = "${FIRST}" ]; then
    pass "/notify ↑ 回到首行"
  else
    fail "/notify ↑ 未回到首行（after=${BACK} first=${FIRST}）"
  fi
fi

# ---- 3) 搜索页结果列表：↓/↑ 逐项移动（search-nav 候选口径）----
ab open "${BASE}/search" >/dev/null 2>&1
COUNT=0
WAIT=0
while [ "${WAIT}" -lt 25 ]; do
  RAW="$(search_rows_json)"
  COUNT="$(qa_json_field "${RAW}" count)"
  [ "${COUNT:-0}" -ge 2 ] && break
  sleep 0.4
  WAIT=$((WAIT + 1))
done
FIRST="$(qa_json_field "${RAW}" first)"
SECOND="$(qa_json_field "${RAW}" second)"
if [ "${COUNT:-0}" -lt 2 ]; then
  fail "/search 结果候选不足 2（页面未加载或选择器失配），无法验证键盘导航"
else
  ab eval '(() => {
    const root = document.querySelector("[data-search-root]");
    const els = [...root.querySelectorAll("a[href], button:not([disabled]), [data-cover]")].filter((el) =>
      !el.closest("header") &&
      !el.closest("[role=\"tablist\"], [data-tab-bar], .input-surface") &&
      el.offsetParent !== null &&
      !el.hasAttribute("aria-hidden")
    );
    els[0]?.focus();
    return true;
  })()' >/dev/null 2>&1
  ab press ArrowDown >/dev/null 2>&1
  AFTER="$(active_label)"
  if [ -n "${SECOND}" ] && [ "${AFTER}" = "${SECOND}" ]; then
    pass "/search ↓ 移到第二个候选（${SECOND}）"
  else
    fail "/search ↓ 未移动（after=${AFTER} first=${FIRST} second=${SECOND}）"
  fi
  ab press ArrowUp >/dev/null 2>&1
  BACK="$(active_label)"
  if [ "${BACK}" = "${FIRST}" ]; then
    pass "/search ↑ 回到第一个候选"
  else
    fail "/search ↑ 未回退（after=${BACK} first=${FIRST}）"
  fi
fi

echo ""
if [ "${FAIL}" -eq 0 ]; then
  say "结果: PASS（通过 ${PASS_COUNT}，跳过 ${SKIP_COUNT}）"
else
  say "结果: FAIL（通过 ${PASS_COUNT}，跳过 ${SKIP_COUNT}）"
fi
say "报告目录：${OUT}"
exit "${FAIL}"
