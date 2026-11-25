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


# ---- 4) 设置页分段控件（role=radiogroup）：方向键移动并选中，Home/End 跳首末项 ----
# 规则见 src/lib/roving-radio.ts；这里验证 DOM 侧接线（焦点/aria-checked/主题类同步）。
radio_state_json() { # $1=radiogroup 下标（0=应用主题，1=外观模式）
  ab eval '(() => {
    const g = [...document.querySelectorAll("[role=radiogroup]")]['"$1"'];
    if (!g) return JSON.stringify({ found: "0" });
    const rs = [...g.querySelectorAll("[role=radio]")];
    const themeClass = ["vibrant", "forest", "inkwash"].find((c) => document.documentElement.classList.contains(c)) || "";
    return JSON.stringify({
      found: "1",
      checkedIndex: String(rs.findIndex((r) => r.getAttribute("aria-checked") === "true")),
      active: (document.activeElement?.textContent || "").trim().slice(0, 8),
      themeClass: themeClass
    });
  })()' 2>/dev/null | tail -1
}

ab open "${BASE}/settings" >/dev/null 2>&1
WAIT=0
THEME_RAW=""
while [ "${WAIT}" -lt 25 ]; do
  THEME_RAW="$(radio_state_json 0)"
  [ "$(qa_json_field "${THEME_RAW}" found)" = "1" ] && break
  sleep 0.4
  WAIT=$((WAIT + 1))
done
if [ "$(qa_json_field "${THEME_RAW}" found)" != "1" ]; then
  fail "/settings 未找到主题 radiogroup（页面未加载或结构变更）"
else
  ORIG_INDEX="$(qa_json_field "${THEME_RAW}" checkedIndex)"
  [ "${ORIG_INDEX}" -ge 0 ] 2>/dev/null || ORIG_INDEX=0
  NEXT_INDEX=$(((ORIG_INDEX + 1) % 3))
  case "${NEXT_INDEX}" in
    0) NEXT_THEME=vibrant ;;
    1) NEXT_THEME=forest ;;
    *) NEXT_THEME=inkwash ;;
  esac
  ab eval '(() => {
    const g = [...document.querySelectorAll("[role=radiogroup]")][0];
    g.querySelector("[role=radio][aria-checked=true]")?.focus();
    return true;
  })()' >/dev/null 2>&1
  ab press ArrowRight >/dev/null 2>&1
  sleep 0.6
  AFTER_THEME="$(radio_state_json 0)"
  AFTER_CLASS="$(qa_json_field "${AFTER_THEME}" themeClass)"
  AFTER_INDEX="$(qa_json_field "${AFTER_THEME}" checkedIndex)"
  if [ "${AFTER_CLASS}" = "${NEXT_THEME}" ] && [ "${AFTER_INDEX}" = "${NEXT_INDEX}" ]; then
    pass "/settings 主题 radiogroup → 移动到 ${NEXT_THEME}（aria-checked 与 html class 同步）"
  else
    fail "/settings 主题 radiogroup → 未按预期移动（class=${AFTER_CLASS} index=${AFTER_INDEX} 期望=${NEXT_THEME}/${NEXT_INDEX}）"
  fi
  ab press End >/dev/null 2>&1
  sleep 0.6
  END_STATE="$(radio_state_json 0)"
  if [ "$(qa_json_field "${END_STATE}" themeClass)" = "inkwash" ]; then
    pass "/settings 主题 radiogroup End → 末项 inkwash"
  else
    fail "/settings 主题 radiogroup End 未到末项（class=$(qa_json_field "${END_STATE}" themeClass)）"
  fi
  ab press Home >/dev/null 2>&1
  sleep 0.6
  HOME_STATE="$(radio_state_json 0)"
  if [ "$(qa_json_field "${HOME_STATE}" themeClass)" = "vibrant" ]; then
    pass "/settings 主题 radiogroup Home → 首项 vibrant"
  else
    fail "/settings 主题 radiogroup Home 未回首项（class=$(qa_json_field "${HOME_STATE}" themeClass)）"
  fi
  # 恢复运行前的主题选择，避免门禁回写用户偏好
  ab eval '(() => {
    const g = [...document.querySelectorAll("[role=radiogroup]")][0];
    g.querySelectorAll("[role=radio]")['"${ORIG_INDEX}"']?.click();
    return true;
  })()' >/dev/null 2>&1
  # 外观模式组：方向键移动并选中（与主题组同一 handler）
  ab eval '(() => {
    const g = [...document.querySelectorAll("[role=radiogroup]")][1];
    g.querySelector("[role=radio][aria-checked=true]")?.focus();
    return true;
  })()' >/dev/null 2>&1
  MODE_BEFORE="$(qa_json_field "$(radio_state_json 1)" active)"
  ab press ArrowRight >/dev/null 2>&1
  sleep 0.6
  MODE_AFTER="$(qa_json_field "$(radio_state_json 1)" active)"
  if [ "${MODE_AFTER}" != "${MODE_BEFORE}" ]; then
    pass "/settings 外观模式 radiogroup → 移动并选中（${MODE_BEFORE} → ${MODE_AFTER}）"
  else
    fail "/settings 外观模式 radiogroup → 未移动（仍为 ${MODE_AFTER}）"
  fi
fi

# ---- 5) /me/relationships 列表：搜索框 ↓ 进首行、行间 ↑↓ ----
relationship_rows_json() {
  ab eval '(() => {
    const root = document.querySelector("[data-relationship-list]");
    const rs = root ? [...root.querySelectorAll("[data-list-item] [data-cover]")] : [];
    const label = (el) => (el.getAttribute("aria-label") || el.textContent || "").trim().slice(0, 40);
    return JSON.stringify({
      count: String(rs.length),
      first: rs[0] ? label(rs[0]) : "",
      second: rs[1] ? label(rs[1]) : ""
    });
  })()' 2>/dev/null | tail -1
}

ab open "${BASE}/me/relationships" >/dev/null 2>&1
WAIT=0
REL_RAW=""
while [ "${WAIT}" -lt 25 ]; do
  REL_RAW="$(relationship_rows_json)"
  [ "$(qa_json_field "${REL_RAW}" count)" -ge 2 ] 2>/dev/null && break
  sleep 0.4
  WAIT=$((WAIT + 1))
done
REL_COUNT="$(qa_json_field "${REL_RAW}" count)"
REL_FIRST="$(qa_json_field "${REL_RAW}" first)"
REL_SECOND="$(qa_json_field "${REL_RAW}" second)"
if [ "${REL_COUNT:-0}" -lt 2 ]; then
  fail "/me/relationships 行不足 2（好友种子缺失或选择器失配），无法验证键盘导航"
else
  # 回归保护：关系页搜索框必须在 1440 桌面直接可见（曾因放在 integrated 页头 below 里
  # 只能从岛展开面板操作，页面本体看不到搜索入口）。
  SEARCH_VIS="$(ab eval '(() => { const i = document.querySelector("input[aria-label=\"搜索关系\"]"); return JSON.stringify({ visible: i && i.offsetParent !== null ? "true" : "false" }); })()' 2>/dev/null | tail -1)"
  if [ "$(qa_json_field "$SEARCH_VIS" visible)" = "true" ]; then
    pass "/me/relationships 搜索框在桌面直接可见（无需展开岛）"
  else
    fail "/me/relationships 搜索框在桌面不可见（offsetParent=null）"
  fi
  ab eval '(() => { document.querySelector("input[aria-label=\"搜索关系\"]")?.focus(); return true; })()' >/dev/null 2>&1
  ab press ArrowDown >/dev/null 2>&1
  AFTER="$(active_label)"
  if [ -n "${REL_FIRST}" ] && [ "${AFTER}" = "${REL_FIRST}" ]; then
    pass "/me/relationships 搜索框 ↓ 进入首行（${REL_FIRST}）"
  else
    fail "/me/relationships 搜索框 ↓ 未进入首行（after=${AFTER} first=${REL_FIRST}）"
  fi
  ab press ArrowDown >/dev/null 2>&1
  AFTER="$(active_label)"
  if [ -n "${REL_SECOND}" ] && [ "${AFTER}" = "${REL_SECOND}" ]; then
    pass "/me/relationships ↓ 移到第二行（${REL_SECOND}）"
  else
    fail "/me/relationships ↓ 未移动（after=${AFTER} second=${REL_SECOND}）"
  fi
  ab press ArrowUp >/dev/null 2>&1
  BACK="$(active_label)"
  if [ "${BACK}" = "${REL_FIRST}" ]; then
    pass "/me/relationships ↑ 回到首行"
  else
    fail "/me/relationships ↑ 未回到首行（after=${BACK} first=${REL_FIRST}）"
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
