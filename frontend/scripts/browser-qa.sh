#!/usr/bin/env bash
# HotSpot 前端浏览器巡检脚本（agent-browser）。
#
# 前置：
#   1. dev 环境已启动（仓库根：docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d）
#   2. 已安装 agent-browser（npm i -g agent-browser && agent-browser install）
#
# 用法：
#   bun run browser:qa              # 桌面 21 页
#   bun run browser:qa:mobile       # 桌面 21 页 + 移动 5 页（390×844）
#   bash scripts/browser-qa.sh --base http://localhost:8889 [--mobile] [--session name]
#
# 输出：<仓库根>/.workwork/browser-check/qa-<时间戳>/（screenshots/ + report.txt）
# 检查项：横向溢出 / 坏图 / h1 数量 / TabBar 数量 / 控制台错误。

set -u

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/lib/qa-common.sh"
find_ab
find_node
# agent-browser 定位（沙盒 wrapper 优先）与 node 定位统一收敛在 scripts/lib/qa-common.sh。

BASE="http://localhost:8888"
SESSION="hotspot-qa-$$"  # PID 后缀：主检出与 worktree 并发跑互不抢同一实例（2026-10-27）
MOBILE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --mobile) MOBILE=1 ;;
    --base) BASE="${2:-$BASE}"; shift ;;
    --session) SESSION="${2:-$SESSION}"; shift ;;
  esac
  shift
done

OUT="${QA_OUT_DIR:-$REPO_ROOT/.workwork/browser-check}/qa-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUT/screenshots"
REPORT="$OUT/report.txt"
echo "HotSpot 浏览器巡检 $(date '+%Y-%m-%d %H:%M:%S')" | tee "$REPORT"
echo "BASE=$BASE SESSION=$SESSION MOBILE=$MOBILE" | tee -a "$REPORT"

audit_js='(() => {
	const vw = document.documentElement.clientWidth;
	const issues = [];
	if (document.documentElement.scrollWidth > vw + 2) issues.push("overflow sw=" + document.documentElement.scrollWidth + " vw=" + vw);
	document.querySelectorAll("img").forEach((im) => {
		const s = im.getAttribute("src") || "";
		if (s && im.complete && im.naturalWidth === 0) issues.push("broken " + s.slice(0, 80));
	});
	const h1 = document.querySelectorAll("h1").length;
	if (h1 !== 1) issues.push("h1=" + h1);
	// 底部主导航 = MobilePortrait/TabletLayout 的 nav[aria-label="主导航"]；
	// 桌面（侧边栏）与手机横屏（无底栏）合法值为 0，故仅记录不做硬断言。
	// 注意 audit_js 整体是 shell 单引号字符串：JS 内一律用双引号（含转义），不得出现单引号。
	return { url: location.pathname, title: document.title, vw, tabBars: document.querySelectorAll("nav[aria-label=\"主导航\"]").length, issues };
})()'

guest_login() {
	ab open "$BASE/" >/dev/null 2>&1
	ab wait --load networkidle >/dev/null 2>&1
	local need_login
	need_login="$(ab eval '(() => document.body.innerText.includes("随便逛逛（访客）"))()' 2>/dev/null | tr -d '"')"
	[ "$need_login" != "true" ] && return 0
	ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("随便逛逛")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
	ab wait 500 >/dev/null 2>&1
	ab eval '(() => { const i = document.querySelector("input[autocomplete=nickname]") || [...document.querySelectorAll("input")].find((x) => (x.placeholder || "").includes("给自己起个名字")); if (!i) return false; Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(i, "QA巡检"); i.dispatchEvent(new Event("input", { bubbles: true })); return true; })()' >/dev/null 2>&1
	ab eval '(() => { const b = [...document.querySelectorAll("button")].find((x) => x.textContent.includes("以访客身份继续")); if (b) b.click(); return !!b; })()' >/dev/null 2>&1
	ab wait --load networkidle >/dev/null 2>&1
}

run_pages() {
	local label="$1"; shift
	local idx=0
	for path in "$@"; do
		idx=$((idx + 1))
		ab open "$BASE$path" >/dev/null 2>&1 || { echo "OPEN FAIL $path" | tee -a "$REPORT"; continue; }
		ab wait --load networkidle >/dev/null 2>&1
		echo "=== $label $path ===" | tee -a "$REPORT"
		ab eval "$audit_js" 2>&1 | tee -a "$REPORT"
		ab screenshot "$OUT/screenshots/$(printf '%02d' "$idx")-${label}-${path//\//_}.png" 2>&1 | tail -1
	done
}

guest_login

# 页面清单与详情页 ID 发现统一收敛在 scripts/lib/qa-common.sh（public API 优先，
# 空则从首页 href fallback，仍空则跳过并记录）。QA 保持 18 页基线 + 详情页。
build_page_lists
QA_PAGES=("${QA_DESKTOP_PAGES[@]}")
[ -n "$DETAIL_MOMENT" ] && QA_PAGES+=("/moments/$DETAIL_MOMENT")
[ -n "$DETAIL_USER" ] && QA_PAGES+=("/users/$DETAIL_USER")
[ -n "$DETAIL_GROUP" ] && QA_PAGES+=("/groups/$DETAIL_GROUP")
run_pages "desktop" "${QA_PAGES[@]}"

if [ "$MOBILE" = "1" ]; then
	ab set viewport 390 844 >/dev/null 2>&1
	MOBILE_PAGES=("/" "/nearby/moments" "/messages" "/map" "/settings")
	run_pages "mobile" "${MOBILE_PAGES[@]}"
fi

echo "=== console errors ===" | tee -a "$REPORT"
ab errors 2>&1 | tee -a "$REPORT" || true
ab close >/dev/null 2>&1

echo "完成：$OUT" | tee -a "$REPORT"
